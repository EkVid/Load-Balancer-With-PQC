package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"lb/models"
	"lb/utils"
	"lb/server"
)

func main() {
	// Start backend servers first
	start := false
	if start {
		numServers := 5
		startingPort := 8081
		for i := 0; i < numServers; i++ {
			port := startingPort + i
			go server.StartServer(port)
			time.Sleep(200 * time.Millisecond) // avoid race
		}
	}

	
	// List of backend URLs
	backendURLs := []string{
		"http://localhost:8081",
		"http://localhost:8082",
		"http://localhost:8083",
		"http://localhost:8084",
		"http://localhost:8085",
	}

	// Create backend objects
	var backends []*models.Backend
	for _, rawURL := range backendURLs {
		backend, err := models.CreateBackendServer(rawURL)
		if err != nil {
			log.Fatalf("Invalid backend URL %s: %v", rawURL, err)
		}
		backends = append(backends, backend)
	}

	// Create load balancer
	lb := &models.LoadBalancer{
		Backends: backends,
	}

	// Start health check
	lb.StartHealthCheck(5 * time.Second)

	utils.StartAdminServer(lb)

	fmt.Println("Load Balancer started on :8080")
	if err := http.ListenAndServe(":8080", lb); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
