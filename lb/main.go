package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync/atomic"
	"time"
)

// Backend represents a single backend server
type Backend struct {
	URL          *url.URL
	Alive        bool
	ReverseProxy *httputil.ReverseProxy
}

// LoadBalancer holds backend pool and counter
type LoadBalancer struct {
	backends []*Backend
	counter  uint64
}

// NewLoadBalancer initializes the load balancer with given backend URLs
func NewLoadBalancer(backendURLs []string) *LoadBalancer {
	var backends []*Backend
	for _, backend := range backendURLs {
		u, err := url.Parse(backend)
		if err != nil {
			log.Fatalf("Invalid backend URL %s: %v", backend, err)
		}
		backends = append(backends, &Backend{
			URL:          u,
			Alive:        true,
			ReverseProxy: httputil.NewSingleHostReverseProxy(u),
		})
	}
	return &LoadBalancer{backends: backends}
}

// getNextBackend returns the next available backend using round-robin
func (lb *LoadBalancer) getNextBackend() *Backend {
	for i := 0; i < len(lb.backends); i++ {
		index := int(atomic.AddUint64(&lb.counter, 1)) % len(lb.backends)
		if lb.backends[index].Alive {
			return lb.backends[index]
		}
	}
	return nil
}

// ServeHTTP implements the http.Handler interface for the load balancer
func (lb *LoadBalancer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	backend := lb.getNextBackend()
	if backend == nil {
		http.Error(w, "Service not available", http.StatusServiceUnavailable)
		return
	}
	fmt.Printf("Forwarding request to: %s\n", backend.URL)
	backend.ReverseProxy.ServeHTTP(w, r)
}

// simpleHealthCheck pings each backend periodically
func (lb *LoadBalancer) simpleHealthCheck() {
	for {
		for _, backend := range lb.backends {
			resp, err := http.Get(backend.URL.String())
			if err != nil || resp.StatusCode >= 400 {
				backend.Alive = false
			} else {
				backend.Alive = true
			}
			if resp != nil {
				resp.Body.Close()
			}
		}
		time.Sleep(5 * time.Second)
	}
}

func main() {
	// List your backend servers here
	backends := []string{
		"http://localhost:8081",
		"http://localhost:8082",
	}

	lb := NewLoadBalancer(backends)

	// Start health check in background
	go lb.simpleHealthCheck()

	fmt.Println("Load Balancer started on :8080")
	if err := http.ListenAndServe(":8080", lb); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
