// this file defines the data structure and some helper functions for the backend server
package models

import (
	"net/http/httputil"
	"net/url"
	"sync"
	"sync/atomic"
	"time"
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
	failCount    int32 // total failure count
	failHistory  []time.Time // timestamps of all failures
	disableUntil time.Time 
}

const ServerMaxLoad = 5 // req to handle simutaneously
const FailurePeriod = 1 // how long (min) for fail history to disable the server
const MaxAllowedTotalFailures = 5
const MaxAllowedFailuresInPeriod = 3
const DisaledPeriodServer = 1

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
		failCount:    0,
	}, nil
}

// load balancing algorithm - weighted + response time + least-loaded 
func (lb *LoadBalancer) GetNextBackend() *Backend {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	var best *Backend
	bestScore := float64(-1 << 62)

	for _, backend := range lb.Backends {
		if !backend.IsAlive() || atomic.LoadInt64(&backend.CurrentLoad) >= backend.MaxLoad || 
		backend.failCount >= MaxAllowedTotalFailures || 
		backend.disableUntil.After(time.Now()) {
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

func (b *Backend) RecentFailureCounts(duration time.Duration) int {
	b.mu.Lock()
    defer b.mu.Unlock()

    cutoff := time.Now().Add(-duration)
    count := 0
    for _, t := range b.failHistory {
        if t.After(cutoff) {
            count++
        }
    }
    return count
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