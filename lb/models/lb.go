// this file defines the data structure and helper functions for LB
package models

import (
	"fmt"
	"net/http"
	"time"
	"sync/atomic"
	"sync"
	"strconv"

	"lb/server"
)

// LoadBalancer represents your load balancer
type LoadBalancer struct {
	Backends []*Backend 
	Counter  uint64    
	mu sync.RWMutex
	MaxConcurrent int64
	ActiveRequests int64
	RateLimiter *RateLimiter
}

func (lb *LoadBalancer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Global congestion control: rate limiting
	if lb.RateLimiter != nil && !lb.RateLimiter.Allow() {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error":"Rate limit exceeded"}`))
		return
	}

	// Global congestion control: max concurrent
	if lb.MaxConcurrent > 0 && atomic.LoadInt64(&lb.ActiveRequests) >= lb.MaxConcurrent {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"error":"Server overloaded"}`))
		return
	}

	atomic.AddInt64(&lb.ActiveRequests, 1)
	defer atomic.AddInt64(&lb.ActiveRequests, -1)

	// Select backend (per-backend congestion aware)
	backend := lb.GetNextBackend()
	if backend == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"error":"No healthy backend available"}`))
		return
	}

	// Increment backend load
	atomic.AddInt64(&backend.CurrentLoad, 1)
	defer atomic.AddInt64(&backend.CurrentLoad, -1)

	start := time.Now()
	fmt.Printf("Forwarding request to: %s\n", backend.URL)

	backend.ReverseProxy.ServeHTTP(w, r)

	// Update backend average response time
	respTime := time.Since(start).Milliseconds()
	old := backend.avgRespTime.Load()
	newAvg := (old + respTime) / 2
	backend.avgRespTime.Store(newAvg)
}

// healthcheck for backend servers
func (lb *LoadBalancer) StartHealthCheck(interval time.Duration) {
	fmt.Println("Health check started...")
	ticker := time.NewTicker(interval)
	go func() {
		client := &http.Client{
			Timeout: 5 * time.Second,
		}
		for range ticker.C {
			for _, backend := range lb.Backends {
				start := time.Now()
				resp, err := client.Get(backend.URL.String())
				elapsed := time.Since(start).Milliseconds()

				if err != nil || resp.StatusCode >= 400 {
					if backend.IsAlive() {
						fmt.Printf("Server %s was up but now is down\n", backend.URL.String())
					}
					backend.SetAlive(false)
					continue
				}

				if !backend.IsAlive() {
					fmt.Printf("Server %s is back up running\n", backend.URL.String())
				}
				backend.SetAlive(true)
				backend.avgRespTime.Store((backend.avgRespTime.Load() + elapsed) / 2)

				if resp != nil {
					resp.Body.Close()
				}
			}
		}
	}()
}

func (lb *LoadBalancer) AddBackend(rawURL string) error {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	backend, err := CreateBackendServer(rawURL)
	if err != nil {
		return err
	}

    // Check if backend is alive before adding
    resp, err := http.Get(backend.URL.String())
    if err != nil || resp.StatusCode >= 400 {
        backend.SetAlive(false)
        fmt.Printf("Added backend %s (currently DOWN)\n", rawURL)
    } else {
        backend.SetAlive(true)
        fmt.Printf("Added backend %s (UP)\n", rawURL)
        resp.Body.Close()
    }

	lb.Backends = append(lb.Backends, backend)
	return nil
}

func (lb *LoadBalancer) RemoveBackend(rawURL string) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	for i, backend := range lb.Backends {
		if backend.URL.String() == rawURL {
			lb.Backends = append(lb.Backends[:i], lb.Backends[i+1:]...)
			fmt.Printf("Removed backend: %s\n", rawURL)
			return
		}
	}
}

func (lb *LoadBalancer) EnableBackend(rawURL string) error {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	for _, backend := range lb.Backends {
		if backend.URL.String() == rawURL {
			if backend.IsAlive() {
				return fmt.Errorf("backend %s is already enabled", rawURL)
			}

			backend.SetAlive(true)

			portNum, err := strconv.Atoi(rawURL[len(rawURL)-4:])
			if err != nil {
				return fmt.Errorf("Invalid port in URL: %s\n", rawURL)
			}
			go server.StartServer(portNum)
			fmt.Printf("Enabled backend: %s\n", rawURL)

			return nil
		}
	}

	return fmt.Errorf("backend %s not found", rawURL)
}


func (lb *LoadBalancer) DisableBackend(rawURL string) error {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	for _, backend := range lb.Backends {
		if backend.URL.String() == rawURL {
			if !backend.IsAlive() {
				return fmt.Errorf("backend %s is already disabled", rawURL)
			}

			backend.SetAlive(false)

			portNum, err := strconv.Atoi(rawURL[len(rawURL)-4:])
			if err != nil {
				return fmt.Errorf("Invalid port in URL: %s\n", rawURL)
			}
			go server.StopServer(portNum)
			fmt.Printf("Disabled backend: %s\n", rawURL)

			return nil
		}
	}

	return fmt.Errorf("backend %s not found", rawURL)
}