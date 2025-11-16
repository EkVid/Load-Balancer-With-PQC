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
	lastScaleUpTime time.Time
	ScaleCooldown time.Duration
	LowPriorityRequestQueue chan *RequestTask
	HighPriorityWaitingQueue chan *RequestTask
}

const (
	ReqLimitScaleUp = 1 // limit to decide if need to scale up 
	NumSeversToAdd = 3 // how many servers to add if needed
	MinServers = 5 // how many servers we should keep minimum
	ClientTimeOut = 5 // how long client will wait for server to respond (sec)
	ExpoMovingAvgCoef = 0.3 // alpha
)

func (lb *LoadBalancer) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	task := &RequestTask{
		W:           w,
		R:           r,
		EnqueueTime: time.Now(),
		Done:        make(chan struct{}),
	}

	// check priority
    priority := r.Header.Get("Priority")

	if priority != "High" {
		select {
		case lb.LowPriorityRequestQueue <-task:
			// <-task.Done
			return
		default: 
		    fmt.Printf("Server busy, 503 returned\n")
			http.Error(w, "Server busy, try again later", http.StatusServiceUnavailable)
			return
		}
	}

	fmt.Printf("High priority request, process first...\n")

	// Rate limiting
	if lb.RateLimiter != nil && !lb.RateLimiter.Allow() {
		select {
		case lb.HighPriorityWaitingQueue <- task:
			fmt.Println("High Priority Request queued due to rate limit\n")
			// <-task.Done // block until processed
			return
		default:
			http.Error(w, "Server busy, try again later", http.StatusServiceUnavailable)
			return
		}
	}

	// Max concurrency
	if lb.MaxConcurrent > 0 && atomic.LoadInt64(&lb.ActiveRequests) >= lb.MaxConcurrent {
		select {
		case lb.HighPriorityWaitingQueue <- task:
			fmt.Println("High Priority Request queued due to max concurrency\n")
			// <-task.Done // block until processed
			return
		default:
			http.Error(w, "Server busy, try again later", http.StatusServiceUnavailable)
			return
		}
	}

	// Process request
	atomic.AddInt64(&lb.ActiveRequests, 1)
	defer atomic.AddInt64(&lb.ActiveRequests, -1)

	lb.handleRequest(w, r)
}


func (lb *LoadBalancer) handleRequest(w http.ResponseWriter, r *http.Request) {
	// Select backend (per-backend congestion aware)
	backend := lb.GetNextBackend()
	if backend == nil {
		go lb.ScaleUp()
		for i := 0; i < 10; i++ {
			time.Sleep(1 * time.Second)
			backend = lb.GetNextBackend()
			if backend != nil {
				break
			}
		}
		if backend == nil {
			fmt.Printf("No backend available...\n")
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

// function to auto scale servers before emergent scaling up
func (lb *LoadBalancer) StartAutoScaler(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			lb.CheckAndScaleUp()
		}
	}()
}


// function to decide if we need to scale up
func (lb *LoadBalancer) CheckAndScaleUp() {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	count := len(lb.Backends)

	if count == 0 {
		return
	}

	// prevent scaling forever
	if time.Since(lb.lastScaleUpTime) < lb.ScaleCooldown {
		return
	}

	serverCapacity := int64(count) * Backend{}.getMaxLoad()

	// Scaling up: high average load
	if lb.ActiveRequests > ReqLimitScaleUp {
		if serverCapacity < lb.MaxConcurrent {
			toAdd := NumSeversToAdd

			remainingCapacity := lb.MaxConcurrent - serverCapacity
			maxPossibleNewServers := int(math.Ceil(float64(remainingCapacity) / float64(Backend{}.getMaxLoad())))
			if toAdd > maxPossibleNewServers {
				toAdd = maxPossibleNewServers
			}

			if toAdd > 0 {
				fmt.Printf("Potential high load detected. Scaling up by %d servers to be safe\n", toAdd)
				go lb.ScaleUp(toAdd)
			}
		} else {
			return
		}
		return
	}
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
		fmt.Printf("Emergent scaling up: need %d extra backends\n", extraBackends)
	}

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

	// Keep at least some servers alive
	if len(lb.Backends) <= MinServers {
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
			fmt.Printf("Scaled down: stopped backend on port %d\n", portNum)
			break // remove one server at a time per check
		}
	}
}

func (lb *LoadBalancer) StartPeriodicScaleDown(interval time.Duration) {
	ticker := time.NewTicker(interval) 
	go func() {
		for range ticker.C {
			lb.ScaleDown()
		}
	}()
}

// healthcheck for backend servers
func (lb *LoadBalancer) StartHealthCheck(interval time.Duration) {
	fmt.Println("Health check started...")
	ticker := time.NewTicker(interval)
	go func() {
		client := &http.Client{
			Timeout: ClientTimeOut * time.Second,
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
				} else {
					if !backend.IsAlive() {
						fmt.Printf("Server %s is back up running\n", backend.URL.String())
					}
				}

				// exponential moving avg
				alpha := ExpoMovingAvgCoef
				old := float64(backend.avgRespTime.Load())
				newAvg := int64((1-alpha)*old + alpha*float64(elapsed))
				backend.avgRespTime.Store(newAvg)

				if resp != nil {
					resp.Body.Close()
				}
			}
		}
	}()
}