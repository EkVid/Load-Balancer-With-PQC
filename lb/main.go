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
	lb := &models.LoadBalancer{
		MaxConcurrent: 20,                                 // global concurrent limit
		RateLimiter:   models.NewRateLimiter(50, time.Second), // 50 req/sec
	}
	
	start := true
	numServers := 5
	startingPort := 8081

	if start {
		// Start backend servers
		for i := 0; i < numServers; i++ {
			port := startingPort + i
			go server.StartServer(port)
			time.Sleep(200 * time.Millisecond)
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

	// Start health checks
	lb.StartHealthCheck(5 * time.Second)

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

	// Start listening for requests
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Printf("Server failed: %v\n", err)
	}

	wg.Wait()
	fmt.Println("Shutdown complete. Exiting program.")
}
