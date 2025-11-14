// this file defines the data structure and some helper functions for the backend server
package models

import (
	"net/http/httputil"
	"net/url"
	"sync"
	"sync/atomic"
	// "time"
	"fmt"
	"net/http"
	"strconv"

	"lb/server"
)

type Backend struct {
	URL          *url.URL
	Alive        bool
	mu           sync.RWMutex 
	ReverseProxy *httputil.ReverseProxy
	Weight 		 int
	avgRespTime  atomic.Int64
	MaxLoad      int64
	CurrentLoad  int64
}

const ServerMaxLoad = 5 // req to handle simutaneously

// creates and initializes a new Backend Server
func CreateBackendServer(rawURL string) (*Backend, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	return &Backend{
		URL:          parsedURL,
		Alive:        true,
		ReverseProxy: httputil.NewSingleHostReverseProxy(parsedURL),
		Weight:       1,    
		MaxLoad:      ServerMaxLoad, 
		CurrentLoad:  0,
		// failCount:    0,
	}, nil
}

// load balancing algorithm - weighted + response time + least-loaded 
func (lb *LoadBalancer) GetNextBackend() *Backend {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	var best *Backend
	bestScore := float64(-1 << 62)

	for _, backend := range lb.Backends {
		if !backend.IsAlive() || atomic.LoadInt64(&backend.CurrentLoad) >= backend.MaxLoad {
			continue
		 }

		respTime := float64(backend.avgRespTime.Load())
		if respTime == 0 {
			respTime = 1 // avoid div by zero
		}

		score := float64(backend.Weight)/respTime - float64(backend.CurrentLoad)
		if score > bestScore {
			bestScore = score
			best = backend
		}
	}

	return best
}

// SetAlive safely updates the Alive status of a backend
func (b *Backend) SetAlive(alive bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.Alive = alive
}

// IsAlive safely checks if the backend is alive
func (b *Backend) IsAlive() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.Alive
}

// return the maxLoad for each backend
func (b Backend) getMaxLoad() int64 {
	return ServerMaxLoad
}

// Add a backend to a LoadBalancer
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

// Remove a backend from a LoadBalancer
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

// Enable a backend
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

// Disable a backend
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