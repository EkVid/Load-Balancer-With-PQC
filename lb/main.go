package main

import (
	"fmt"
	"net/http"
	"time"
	"sync"

	"log"
	"strconv"

	"lb/config"
	"lb/models"
	"lb/server"
	"lb/utils"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig("config/config.json")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// setting params
	start := cfg.StartServers
	numServers := cfg.NumServers
	startingPort := cfg.StartingPort
	scaleCooldown := time.Duration(cfg.ScaleCooldownSeconds) * time.Second
	lowqueueLength := cfg.LowPriorityQueueLength
	highqueueLength := cfg.HighPriorityQueueLength
	maxQueueingTime := cfg.MaxQueueingTimeSeconds
	maxConcurrent := cfg.MaxConcurrentRequests
	rateLimit := cfg.RateLimitPerSecond
	checkingPeriod := time.Duration(cfg.HealthCheckPeriodSeconds) * time.Second
	scaleDownPeriod := time.Duration(cfg.ScaleDownPeriodSeconds) * time.Second
	serverStartingDelay := cfg.ServerStartingDelayMs

	lb := &models.LoadBalancer{
		// global concurrent limit (How many active requests can the server handle concurrently before rejecting or queueing)
		MaxConcurrent: int64(maxConcurrent),                          
		RateLimiter:   models.NewRateLimiter(rateLimit, time.Second), // total number of requests allowed per second
		NextPort: startingPort,
		ScaleCooldown: scaleCooldown,
		LowPriorityRequestQueue: make(chan *models.RequestTask, lowqueueLength),
		HighPriorityWaitingQueue: make(chan *models.RequestTask, highqueueLength),
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
				fmt.Printf("Invalid backend URL %s: %v\n", rawURL, err)
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

	// Main LB HTTP server
	srv := &http.Server{
		Addr:    ":" + strconv.Itoa(cfg.LBPort),
		Handler: lb,
	}

	fmt.Printf("Load Balancer started on :%d\n", cfg.LBPort)

	// Handle graceful shutdown (in a goroutine so it doesn't block)
	var wg sync.WaitGroup
	wg.Add(1)
	go utils.GracefulShutdown(lb, srv, &wg)

	// Start a background worker that pulls from the queue
	lb.StartQueueProcessor(time.Duration(maxQueueingTime) * time.Second, 20)

	// Start listening for requests
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Printf("Server failed: %v\n", err)
	}

	wg.Wait()
	fmt.Println("Shutdown complete. Exiting program\n")
}

// TODO: add decryption logic when accepting HTTPS with PQC