// this file defines the data structure and some helper functions for the backend server
package models

import (
	"net/http/httputil"
	"net/url"
	"sync"
	"sync/atomic"
)

type Backend struct {
	URL          *url.URL
	Alive        bool
	mu           sync.RWMutex 
	ReverseProxy *httputil.ReverseProxy
	Weight 		 int
	currentLoad  int64
	avgRespTime  atomic.Int64
	MaxLoad      int64
	CurrentLoad  int64
}

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
		MaxLoad:      5, 
		CurrentLoad:  0,
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
