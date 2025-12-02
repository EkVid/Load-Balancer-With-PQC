package models

import (
	"net/http"
	"log"
	"sync/atomic"
	"time"
	"math/rand"

	"lb/config"
)
// RequestTask holds the queued HTTP request
type RequestTask struct {
	W       http.ResponseWriter
	R       *http.Request
	EnqueueTime time.Time
	Done    chan struct{} // when a request is done
}

// StartQueueProcessor starts a background worker that processes queued requests
func (lb *LoadBalancer) StartQueueProcessor(timeout time.Duration, workerCount int) {
	for i := 0; i < workerCount; i++ {
		go func(id int) {
			highWeight := config.HighWeight
			lowWeight := config.LowWeight
			for {
				total := highWeight + lowWeight
				choice := rand.Intn(total) // 0..total-1
	
				if choice < highWeight && len(lb.HighPriorityWaitingQueue) > 0 {
					// process one high-priority task
					lb.ProcessQueue(lb.HighPriorityWaitingQueue, timeout, config.MaxTasks)
				} else if len(lb.LowPriorityRequestQueue) > 0 {
					// process one low-priority task
					lb.ProcessQueue(lb.LowPriorityRequestQueue, timeout, config.MaxTasks)
				}
	
				// small sleep to avoid busy loop
				time.Sleep(config.SleepTime * time.Millisecond)
			}
		}(i)
	}
}


func (lb *LoadBalancer) ProcessQueue(q chan *RequestTask, timeout time.Duration, maxTasks int) {
	count := 0
	for task := range q {
		if maxTasks > 0 && count >= maxTasks {
			break
		}
		count++

		// Wait if still at max concurrency
		for lb.MaxConcurrent > 0 && atomic.LoadInt64(&lb.ActiveRequests) >= lb.MaxConcurrent {
			time.Sleep(config.SleepTimeMaxConcurrency * time.Millisecond)
		}

		// Drop if request exceeded queue timeout
		if time.Since(task.EnqueueTime) > timeout {
			if task.W != nil {
				log.Println("Dropping queued request due to timeout")
				http.Error(task.W, "Request timed out in queue", http.StatusServiceUnavailable)
			}
			if task.Done != nil {
				close(task.Done)
			}
			continue
		}

		backend := lb.GetNextBackend()
		if backend == nil {
			log.Println("No backend available for queued request, requeueing")
			select {
			case q <- task:
				time.Sleep(config.SleepTimeQueue * time.Millisecond)
			default:
				log.Println("Queue full, dropping queued request")
				if task.Done != nil {
					close(task.Done)
				}
			}
			continue
		}

		atomic.AddInt64(&lb.ActiveRequests, 1)
		lb.handleRequest(task.W, task.R)
		atomic.AddInt64(&lb.ActiveRequests, -1)

		if task.Done != nil {
			close(task.Done)
		}
	}
}
