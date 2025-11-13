// this file defines the data structure and helper functions for LB
package models

import (
	"fmt"
	"math"
	"net/http"
	"time"
	"sync/atomic"
	"sync"
	"strconv"

	"lb/server"
)

// LoadBalancer struct
type LoadBalancer struct {
	Backends []*Backend 
	Counter  uint64    
	mu sync.RWMutex
	MaxConcurrent int64
	ActiveRequests int64
	RateLimiter *RateLimiter
	NextPort int
}

func (lb *LoadBalancer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Global congestion control: rate limiting
	if lb.RateLimiter != nil && !lb.RateLimiter.Allow() {
		fmt.Printf("Number of requests exceeding the total rate\n")
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error":" Global Rate limit exceeded"}`))
		return
	}

	// Global congestion control: max concurrent
	if lb.MaxConcurrent > 0 && atomic.LoadInt64(&lb.ActiveRequests) >= lb.MaxConcurrent {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"error":" Server overloaded"}`))
		return
	}

	atomic.AddInt64(&lb.ActiveRequests, 1)
	defer atomic.AddInt64(&lb.ActiveRequests, -1)

	// Select backend (per-backend congestion aware)
	backend := lb.GetNextBackend()
	if backend == nil {
		// emergency scaling up
		go lb.ScaleUp()

		// wait a few seconds then retry 
		for i := 0; i < 10; i++ {
			time.Sleep(500 * time.Millisecond)
			backend = lb.GetNextBackend()
			if backend != nil {
				fmt.Println("Backend became available, proceeding...")
				break
			}
		}

		if backend == nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"error":"Scaling up, please retry later"}`))
			return
		}
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

// function to decide scale up or down
func (lb *LoadBalancer) CheckAndScale() {
	totalLoad := int64(0)
	lb.mu.RLock()
	for _, backend := range lb.Backends {
		totalLoad += atomic.LoadInt64(&backend.CurrentLoad)
	}
	count := len(lb.Backends)
	lb.mu.RUnlock()

	if count == 0 {
		return
	}

	// TODO: fix the logic for avg load, it's printing 0 consistently
	avgLoad := float64(totalLoad) / float64(count)
	serverCapacity := int64(count) * Backend{}.getMaxLoad()
	fmt.Printf("Avg load is: %f\n", avgLoad)

	// Scaling up: high average load
	if avgLoad > 3 {
		if serverCapacity < lb.MaxConcurrent {
			toAdd := 3

			remainingCapacity := lb.MaxConcurrent - serverCapacity
			maxPossibleNewServers := int(math.Ceil(float64(remainingCapacity) / float64(Backend{}.getMaxLoad())))
			if toAdd > maxPossibleNewServers {
				toAdd = maxPossibleNewServers
			}

			if toAdd > 0 {
				fmt.Printf("High load detected (avg=%.2f). Scaling up by %d servers.\n", avgLoad, toAdd)
				go lb.ScaleUp(toAdd)
			}
		} else {
			fmt.Println("Already at max global capacity, cannot scale up further.")
		}
		return
	}

	// Scaling down: low load
	lb.ScaleDown()
}


// add and enable more servers to handle the traffics (takes optional input to decide how many servers to scale up)
func (lb *LoadBalancer) ScaleUp(optionalCount ...int) {
    lb.mu.Lock()
    defer lb.mu.Unlock()

	var extraBackends int
	// if arg passed in
	if len(optionalCount) > 0 {
		extraBackends = optionalCount[0]
	} else {
		// default behavior
		maxLoad := int(Backend{}.getMaxLoad())
		extraBackends = int(math.Ceil(float64(lb.MaxConcurrent - int64(maxLoad*len(lb.Backends))) / float64(maxLoad)))
	}

	fmt.Printf("Scaling up: need %d extra backends\n", extraBackends)
    for i := 0; i < extraBackends; i++ {
        port := lb.NextPort
        templateURL := fmt.Sprintf("http://localhost:%d", port)

        // Create backend struct
		backend, err := CreateBackendServer(templateURL)
		if err != nil {
			fmt.Printf("Error creating server\n")
		}

        // Add to main backend list
        lb.Backends = append(lb.Backends, backend)
        // Start server asynchronously
        go func(p int) {
            server.StartServer(p)
        }(port)

        lb.NextPort++
    }
}

// remove and disable some servers 
func (lb *LoadBalancer) ScaleDown() {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	if len(lb.Backends) <= 5 {
		// Keep at least 5 servers alive
		return
	}

	// Only consider servers with no active load
	for i := len(lb.Backends) - 1; i >= 0; i-- {
		backend := lb.Backends[i]
		if atomic.LoadInt64(&backend.CurrentLoad) == 0 {
			portNum, err := strconv.Atoi(backend.URL.Port())
			if err != nil {
				fmt.Printf("Invalid port for backend %s, skipping\n", backend.URL.String())
				continue
			}

			server.StopServer(portNum)
			lb.Backends = append(lb.Backends[:i], lb.Backends[i+1:]...)
			fmt.Printf("Scaled down safely: stopped backend on port %d\n", portNum)
			break // remove one server at a time per check
		}
	}
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

				// exponential moving avg
				alpha := 0.3
				old := float64(backend.avgRespTime.Load())
				newAvg := int64((1-alpha)*old + alpha*float64(elapsed))
				backend.avgRespTime.Store(newAvg)

				if resp != nil {
					resp.Body.Close()
				}
			}
			// periodically check and scale as well
			lb.CheckAndScale()
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