// this file defines the data structure and some helper functions for the backend server
package models

import (
	"net/http/httputil"
	"net/url"
	"sync"
)

type Backend struct {
	URL          *url.URL
	Alive        bool
	mu           sync.RWMutex // Protects access to Alive
	ReverseProxy *httputil.ReverseProxy
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
	}, nil
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
