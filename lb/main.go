package main

import (
	"fmt"
	"log"
	"net/http"
	"time"
	"sync"
	"strconv"
	"strings"

	"lb/config"
	"lb/models"
	"lb/server"
	"lb/utils"
)

func main() {
	// setting params
	lbPort := config.LBPort
	start := config.StartServers
	numServers := config.NumServers
	startingPort := config.StartingPort
	scaleCooldown := time.Duration(config.ScaleCooldownSeconds) * time.Second
	lowQueueLength := config.LowPriorityQueueLength
	highQueueLength := config.HighPriorityQueueLength
	maxQueueingTime := config.MaxQueueingTimeSeconds
	maxConcurrent := config.MaxConcurrentRequests
	rateLimit := config.RateLimitPerSecond
	checkingPeriod := time.Duration(config.HealthCheckPeriodSeconds) * time.Second
	scaleDownPeriod := time.Duration(config.ScaleDownPeriodSeconds) * time.Second
	serverStartingDelay := config.ServerStartingDelayMs
	numWorkers := config.NumberOfWorkers

	lb := &models.LoadBalancer{
		MaxConcurrent: int64(maxConcurrent), // global concurrent limit (How many active requests can the server handle concurrently before rejecting or queueing)                        
		RateLimiter:   models.NewRateLimiter(rateLimit, time.Second), // total number of requests allowed per second
		NextPort: startingPort,
		ScaleCooldown: scaleCooldown,
		LowPriorityRequestQueue: make(chan *models.RequestTask, lowQueueLength),
		HighPriorityWaitingQueue: make(chan *models.RequestTask, highQueueLength),
	}

	if start {
		// Start backend servers
		for i := 0; i < numServers; i++ {
			port := startingPort + i
			go server.StartServer(port)
			time.Sleep(time.Duration(serverStartingDelay) * time.Millisecond)
		}

		// Create and populate backend objects
		for i := 0; i < numServers; i++ {
			port := startingPort + i
			rawURL := fmt.Sprintf("http://localhost:%d", port)

			backend, err := models.CreateBackendServer(rawURL)
			if err != nil {
				log.Printf("Invalid backend URL %s: %v\n", rawURL, err)
				continue
			}
			lb.Backends = append(lb.Backends, backend)
		}
	}

	// next available port
	lb.NextPort = startingPort + numServers

	// Start health checks to run periodically
	lb.StartHealthCheck(checkingPeriod)
	
	// Start auto scaler to run periodically
	lb.StartAutoScaler(checkingPeriod)

	// start scale down
	lb.StartPeriodicScaleDown(scaleDownPeriod)

	// Start admin API
	utils.StartAdminServer(lb)

	// Start a background worker that pulls from the queue
	lb.StartQueueProcessor(time.Duration(maxQueueingTime) * time.Second, numWorkers)

	// Handle graceful shutdown
	var wg sync.WaitGroup

	// If HTTPS is enabled
	if config.EnableHTTPS {
		// Main HTTPS server with secure TLS configuration
		httpsSrv := &http.Server{
			Addr:      ":" + strconv.Itoa(config.HTTPSPort),
			Handler:   lb,
			TLSConfig: utils.GetSecureTLSConfig(),
		}

		log.Printf("HTTPS Load Balancer started on :%d\n", config.HTTPSPort)

		// Optional: Redirect HTTP to HTTPS
		if config.RedirectHTTPToHTTPS {
			httpSrv := &http.Server{
				Addr: ":" + strconv.Itoa(config.HTTPPort),
				Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					host := r.Host
					if colonPos := strings.Index(host, ":"); colonPos != -1 {
                		host = host[:colonPos]
					}

					httpsURL := fmt.Sprintf("https://%s:%d%s", host, config.HTTPSPort, r.RequestURI)
					http.Redirect(w, r, httpsURL, http.StatusMovedPermanently)
				}),
			}

			log.Printf("HTTP redirect server started on :%d (redirecting to HTTPS)\n", config.HTTPPort)

			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					log.Printf("HTTP redirect server failed: %v\n", err)
				}
			}()

			wg.Add(1)
			go utils.GracefulShutdown(lb, httpSrv, &wg)
		}

		wg.Add(1)
		go utils.GracefulShutdown(lb, httpsSrv, &wg)

		// Start HTTPS listener
		if err := httpsSrv.ListenAndServeTLS(config.CertFile, config.KeyFile); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTPS server failed: %v\n", err)
		}
	} else {
		// Main HTTP server (original behavior)
		srv := &http.Server{
			Addr:    ":" + strconv.Itoa(lbPort),
			Handler: lb,
		}

		log.Printf("HTTP Load Balancer started on :%d\n", lbPort)

		wg.Add(1)
		go utils.GracefulShutdown(lb, srv, &wg)

		// Start listening for requests
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server failed: %v\n", err)
		}
	}

	wg.Wait()
	log.Println("Shutdown complete. Exiting program\n")
}

