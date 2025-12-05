package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
	"math/rand"
	"lb/config"
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

var failureCount int64
var highFailure int64
var lowFailure int64

var blockedCount int64
var highBlocked int64
var lowBlocked int64



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
	if rand.Intn(2) == 1 { // 50% chance to set it
		req.Header.Set("X-Forwarded-For", randomIP())
	}
}

func countRequests(status int, priority string) {
	switch status {
	case 200:
		atomic.AddInt64(&successCount, 1)
		if priority == "High" {
			atomic.AddInt64(&highSuccess, 1)
		} else {
			atomic.AddInt64(&lowSuccess, 1)
		}

	case 403:
		atomic.AddInt64(&blockedCount, 1)
		if priority == "High" {
			atomic.AddInt64(&highBlocked, 1)
		} else {
			atomic.AddInt64(&lowBlocked, 1)
		}

	default:
		atomic.AddInt64(&failureCount, 1)
		if priority == "High" {
			atomic.AddInt64(&highFailure, 1)
		} else {
			atomic.AddInt64(&lowFailure, 1)
		}
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

	// comment out if dont want firewall
	maybeAddForwardedFor(req)

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Request error:", err)
		// Count as failure
		atomic.AddInt64(&failureCount, 1)
		if priority == "High" {
			atomic.AddInt64(&highFailure, 1)
		} else {
			atomic.AddInt64(&lowFailure, 1)
		}
		return
	}
	defer resp.Body.Close()

	fromIP := req.Header.Get("X-Forwarded-For")
	if fromIP == "" {
		fromIP = "N/A"
	}
	
	fmt.Printf("Priority: %s, Status: %d, From: %s\n",
		priority, resp.StatusCode, fromIP)
	
	// Count requests
	countRequests(resp.StatusCode, priority)
}

func main() {
	rand.Seed(time.Now().UnixNano())

	var client *http.Client
	if config.EnableHTTPS {
		client = &http.Client{
            Transport: &http.Transport{
                TLSClientConfig: &tls.Config{
                    InsecureSkipVerify: true, // Skip certificate verification for testing
                },
            },
        }
	}else {
		client = &http.Client{}
	}
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
	fmt.Printf("\nAll requests completed.\n")
	fmt.Printf("Total: %d\n", totalRequests)
	fmt.Printf("Successful (200): %d\n", successCount)
	fmt.Printf("Blocked (403): %d\n", blockedCount)
	fmt.Printf("Failed (others): %d\n", failureCount)

	fmt.Printf("\nHigh priority requests sent: %d\n", highCount)
	fmt.Printf("    Successful: %d\n", highSuccess)
	fmt.Printf("    Blocked:    %d\n", highBlocked)
	fmt.Printf("    Failed:     %d\n", highFailure)

	fmt.Printf("\nLow priority requests sent: %d\n", lowCount)
	fmt.Printf("    Successful: %d\n", lowSuccess)
	fmt.Printf("    Blocked:    %d\n", lowBlocked)
	fmt.Printf("    Failed:     %d\n", lowFailure)

}
