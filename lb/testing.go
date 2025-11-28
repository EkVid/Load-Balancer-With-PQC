package main

import (
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
	"math/rand"
)

const (
	// change here for how many req to send and how many to send concurrently
	totalRequests  = 300
	maxConcurrency = 100
	url            = "http://localhost:8080"
)

var successCount int64
var highCount int64
var lowCount int64
var highSuccess int64
var lowSuccess int64

var ipList = []string{
	"203.0.113.42",
	"123.45.67.89",
	"10.20.30.40",
	"203.0.113.99",
}

func randomIP() string {
	return ipList[rand.Intn(len(ipList))]
}

func maybeAddForwardedFor(req *http.Request) {
	if rand.Intn(2) == 1 { // 50% chance
		req.Header.Set("X-Forwarded-For", randomIP())
	}
}


func sendRequest(client *http.Client, priority string, wg *sync.WaitGroup, sem chan struct{}) {
	defer wg.Done()

	// Count how many of each priority were sent
	if priority == "High" {
		atomic.AddInt64(&highCount, 1)
	} else {
		atomic.AddInt64(&lowCount, 1)
	}

	// Acquire semaphore
	sem <- struct{}{}
	defer func() { <-sem }()

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println("Error creating request:", err)
		return
	}

	req.Header.Set("Priority", priority)

	maybeAddForwardedFor(req)

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Request error:", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("Priority: %s, Status: %d\n", priority, resp.StatusCode)

	// Count successful requests
	if resp.StatusCode == 200 {
		atomic.AddInt64(&successCount, 1)
		if priority == "High" {
			atomic.AddInt64(&highSuccess, 1)
		} else {
			atomic.AddInt64(&lowSuccess, 1)
		}
	}
}

func main() {
	rand.Seed(time.Now().UnixNano())

	client := &http.Client{}
	wg := sync.WaitGroup{}

	// Semaphore channel to limit concurrency
	sem := make(chan struct{}, maxConcurrency)

	for i := 0; i < totalRequests; i++ {
		wg.Add(1)

		// Randomly choose priority
		priority := "Low"
		if rand.Intn(2) == 1 {
			priority = "High"
		}

		go sendRequest(client, priority, &wg, sem)
	}

	// Wait for all requests to finish
	wg.Wait()
	fmt.Printf("All requests completed.\n")
	fmt.Printf("Successful (200) requests: %d/%d\n", successCount, totalRequests)
	fmt.Printf("High priority requests sent: %d, Successful: %d\n", highCount, highSuccess)
	fmt.Printf("Low priority requests sent: %d, Successful: %d\n", lowCount, lowSuccess)
}
