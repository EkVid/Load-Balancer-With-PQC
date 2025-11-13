package main

import (
	"fmt"
	"net/http"
	"time"
	"sync"

	"lb/models"
	"lb/server"
	"lb/utils"
)

func main() {
	start := true
	numServers := 5
	startingPort := 8081
	scaleCooldown := 10 * time.Second
	queueLength := 100
	maxQueueingTime := 5
	maxConcurrent := 100
	rateLimit := 100
	checkingPeriod := 3 * time.Second
	serverStartingDelay := 200

	lb := &models.LoadBalancer{
		MaxConcurrent: int64(maxConcurrent),                                 // global concurrent limit 100 req (num req that can be handled concurrently)
		RateLimiter:   models.NewRateLimiter(rateLimit, time.Second), // 100 req/sec
		NextPort: startingPort,
		ScaleCooldown: scaleCooldown,
		RequestQueue: make(chan *models.RequestTask, queueLength),
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

	// Start admin API
	utils.StartAdminServer(lb)

	// Main LB HTTP server
	srv := &http.Server{
		Addr:    ":8080",
		Handler: lb,
	}

	fmt.Println("Load Balancer started on :8080\n")

	// Handle graceful shutdown (in a goroutine so it doesn't block)
	var wg sync.WaitGroup
	wg.Add(1)
	go utils.GracefulShutdown(lb, srv, &wg)

	// Start a background worker that pulls from the queue
	lb.StartQueueProcessor(time.Duration(maxQueueingTime) * time.Second)

	// Start listening for requests
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Printf("Server failed: %v\n", err)
	}

	wg.Wait()
	fmt.Println("Shutdown complete. Exiting program\n")
}



