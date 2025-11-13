package models

import (
	"fmt"
	"net/http"
	"sync/atomic"
	"time"
)
// RequestTask holds the queued HTTP request
type RequestTask struct {
	W       http.ResponseWriter
	R       *http.Request
	EnqueueTime time.Time
	Done    chan struct{} // when a request is done
}

// StartQueueProcessor starts a background worker that processes queued requests
func (lb *LoadBalancer) StartQueueProcessor(timeout time.Duration) {
	go func() {
		for task := range lb.RequestQueue {
			// Drop if request exceeded queue timeout
			if time.Since(task.EnqueueTime) > timeout {
				fmt.Println("Dropping queued request due to timeout")
				if task.Done != nil {
					close(task.Done)
				}
				continue
			}

			// Wait if still at max concurrency
			for lb.MaxConcurrent > 0 && atomic.LoadInt64(&lb.ActiveRequests) >= lb.MaxConcurrent {
				time.Sleep(10 * time.Millisecond)
			}

			backend := lb.GetNextBackend()
			if backend == nil {
				fmt.Println("No backend available for queued request, requeueing")
				select {
				case lb.RequestQueue <- task:
					time.Sleep(50 * time.Millisecond)
				default:
					fmt.Println("Queue full, dropping queued request")
					if task.Done != nil {
						close(task.Done)
					}
				}
				continue
			}

			atomic.AddInt64(&lb.ActiveRequests, 1)
			lb.handleRequest(task.W, task.R)
			atomic.AddInt64(&lb.ActiveRequests, -1)

			// Signal handler that request is finished
			if task.Done != nil {
				close(task.Done)
			}
		}
	}()
}


// EnqueueRequest tries to add a request to the queue
func (lb *LoadBalancer) EnqueueRequest(task *RequestTask) bool {
	task.EnqueueTime = time.Now()
	select {
	case lb.RequestQueue <- task:
		fmt.Println("Request queued due to high load")
		return true
	default:
		fmt.Println("Queue full, request dropped")
		return false
	}
}
