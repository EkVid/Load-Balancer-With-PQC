// this file defines the data structure and helper functions for LB
package models

import (
	"fmt"
	"net/http"
	"time"
	"sync/atomic"
	"sync"
)

// LoadBalancer represents your load balancer
type LoadBalancer struct {
	Backends []*Backend // slice of backend servers
	Counter  uint64     // for round-robin
	mu sync.RWMutex
}

// Round-robin
func (lb *LoadBalancer) GetNextBackend() *Backend {
    for i := 0; i < len(lb.Backends); i++ {
        index := int(atomic.AddUint64(&lb.Counter, 1)) % len(lb.Backends)
        if lb.Backends[index].IsAlive() {
            return lb.Backends[index]
        }
    }
    return nil
}

// http.Handler
func (lb *LoadBalancer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    backend := lb.GetNextBackend()
    if backend == nil {
		if backend == nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			jsonResp := `{"error": "Service unavailable", "message": "No healthy backend servers are available."}`
			_, _ = w.Write([]byte(jsonResp))
			return
		}
    }
    fmt.Printf("Forwarding request to: %s\n", backend.URL)
    backend.ReverseProxy.ServeHTTP(w, r)
}


func (lb *LoadBalancer) StartHealthCheck(interval time.Duration) {
	fmt.Println("Health check started...")
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			for _, backend := range lb.Backends {
				resp, err := http.Get(backend.URL.String())
				if err != nil || resp.StatusCode >= 400 {
					if backend.IsAlive() {
						fmt.Printf("Server %s was up but now is down\n", backend.URL.String())
					}
					backend.SetAlive(false)
				} else {
					if !backend.IsAlive() {
						fmt.Printf("Server %s is back up running\n", backend.URL.String())
					}
					backend.SetAlive(true)
					if resp != nil {
						resp.Body.Close()
					}
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
