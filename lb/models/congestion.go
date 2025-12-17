// this file defines the congestion control structure and some functions
package models

import (
	"sync"
	"time"
)

// Simple rate limiter using the token bucket algorithm
type RateLimiter struct {
	rate      int           // allowed requests per interval
	interval  time.Duration // refill interval
	tokens    int
	lastCheck time.Time
	mu        sync.Mutex
}

func NewRateLimiter(rate int, interval time.Duration) *RateLimiter {
	return &RateLimiter{
		rate:      rate,
		interval:  interval,
		tokens:    rate,
		lastCheck: time.Now(),
	}
}

func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(rl.lastCheck)

	// refill tokens
	if elapsed >= rl.interval {
		rl.tokens = rl.rate
		rl.lastCheck = now
	}

	if rl.tokens > 0 {
		rl.tokens--
		return true
	}

	return false
}
