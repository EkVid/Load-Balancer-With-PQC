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
	start := false
	numServers := 5
	startingPort := 8081

	lb := &models.LoadBalancer{
		MaxConcurrent: 100,                                 // global concurrent limit 100 req (num req that can be handled concurrently)
		RateLimiter:   models.NewRateLimiter(100, time.Second), // 100 req/sec
		NextPort: startingPort,
	}

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

	// next available port
	lb.NextPort = startingPort + numServers + 1

	// Start health checks
	lb.StartHealthCheck(5 * time.Second)


    // Start automatic cleanup of temporary backends if there are any for every minute
	// go func() {
	// 	fmt.Println("Starting temporary backend cleanup ticker...")
	// 	ticker := time.NewTicker(5 * time.Second)
	// 	for range ticker.C {
	// 		fmt.Println("Running cleanup...")
	// 		lb.RemoveTempServers()
	// 	}
	// }()
	

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



// Q: if I remove a port, should the next port take over it or its fine to not use it??