
- The client sends a request to the proxy instead of directly to the server.  
- The proxy forwards it to one of several backend servers.  
- The proxy returns the backend’s response to the client.

When multiple backends are involved, it can also **balance the load** using strategies like:
- Round Robin
- Random
- Least Connections
- Weighted Distribution

---

## Why Go Makes This Easy

Go’s standard library (`net/http` and `net/http/httputil`) already includes:
- `http.Server` — for handling incoming client requests  
- `httputil.ReverseProxy` — for forwarding requests to backends  

This means you don’t need to reinvent the HTTP stack.  
You can focus on **routing logic**, **health checks**, and **resilience**.

---

## The Simplest Reverse Proxy in Go

Let’s start small — a reverse proxy that forwards all traffic to a single backend.

```go
package main

import (
	"net/http"
	"net/http/httputil"
	"net/url"
)

func main() {
	target, _ := url.Parse("http://localhost:8081")
	proxy := httputil.NewSingleHostReverseProxy(target)

	http.ListenAndServe(":8080", proxy)
}

---
## Add Load Balancing to scale to multiple backend servers

package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync/atomic"
)

type LoadBalancer struct {
	backends []*url.URL
	counter  uint64
}

func (lb *LoadBalancer) getNextBackend() *url.URL {
	index := atomic.AddUint64(&lb.counter, 1)
	return lb.backends[index%uint64(len(lb.backends))]
}

func (lb *LoadBalancer) handler(w http.ResponseWriter, r *http.Request) {
	target := lb.getNextBackend()
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.ServeHTTP(w, r)
}

func main() {
	backends := []*url.URL{
		{Scheme: "http", Host: "localhost:8081"},
		{Scheme: "http", Host: "localhost:8082"},
		{Scheme: "http", Host: "localhost:8083"},
	}

	lb := &LoadBalancer{backends: backends}

	log.Println("Load balancer running on :8080")
	http.ListenAndServe(":8080", http.HandlerFunc(lb.handler))
}


---
## Add health check to avoid sending traffics to dead servers
package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"sync/atomic"
	"time"
)

type Backend struct {
	URL    *url.URL
	Alive  bool
	mu     sync.RWMutex
	Proxy  *httputil.ReverseProxy
}

type LoadBalancer struct {
	backends []*Backend
	counter  uint64
}

func (b *Backend) setAlive(alive bool) {
	b.mu.Lock()
	b.Alive = alive
	b.mu.Unlock()
}

func (b *Backend) isAlive() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.Alive
}

func (lb *LoadBalancer) getNextBackend() *Backend {
	length := len(lb.backends)
	for i := 0; i < length; i++ {
		index := int(atomic.AddUint64(&lb.counter, 1) % uint64(length))
		if lb.backends[index].isAlive() {
			return lb.backends[index]
		}
	}
	return nil
}

func (lb *LoadBalancer) healthCheck() {
	for {
		for _, backend := range lb.backends {
			resp, err := http.Get(backend.URL.String() + "/health")
			alive := err == nil && resp.StatusCode == http.StatusOK
			backend.setAlive(alive)
			if alive {
				log.Printf("✔ %s is alive\n", backend.URL)
			} else {
				log.Printf("✖ %s is down\n", backend.URL)
			}
		}
		time.Sleep(10 * time.Second)
	}
}

func (lb *LoadBalancer) handler(w http.ResponseWriter, r *http.Request) {
	backend := lb.getNextBackend()
	if backend == nil {
		http.Error(w, "No healthy backends available", http.StatusServiceUnavailable)
		return
	}
	backend.Proxy.ServeHTTP(w, r)
}

func main() {
	backendURLs := []string{
		"http://localhost:8081",
		"http://localhost:8082",
		"http://localhost:8083",
	}

	var backends []*Backend
	for _, rawURL := range backendURLs {
		url, _ := url.Parse(rawURL)
		backends = append(backends, &Backend{
			URL:   url,
			Alive: true,
			Proxy: httputil.NewSingleHostReverseProxy(url),
		})
	}

	lb := &LoadBalancer{backends: backends}
	go lb.healthCheck()

	log.Println("Load balancer started on :8080")
	http.ListenAndServe(":8080", http.HandlerFunc(lb.handler))
}



