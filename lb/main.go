package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
	"lb/models"
)

func main() {
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
		u, err := url.Parse(rawURL)
		if err != nil {
			log.Fatalf("Invalid backend URL %s: %v", rawURL, err)
		}
		backends = append(backends, &models.Backend{
			URL:          u,
			Alive:        true,
			ReverseProxy: httputil.NewSingleHostReverseProxy(u),
		})
	}

	// Create load balancer
	lb := &models.LoadBalancer{
		Backends: backends,
	}

	// Start health check
	lb.StartHealthCheck(5 * time.Second)

	fmt.Println("Load Balancer started on :8080")
	if err := http.ListenAndServe(":8080", lb); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
