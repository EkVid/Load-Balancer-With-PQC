// this file defines the functions for the servers

package server

import (
    "fmt"
    "log"
    "net/http"
    "strconv"
    "time"
    "context"
    "sync"
)

var (
    servers   = make(map[int]*http.Server)
    serversMu sync.Mutex
)

func StartServer(port int) {
	mux := http.NewServeMux()
    mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        // simulate busy server for testing
		// if port == 8083 {
		// 	time.Sleep(10 * time.Second)
		// } else {
		// 	time.Sleep(500 * time.Millisecond)
		// }
        priority := r.Header.Get("Priority")
        if priority != "High" {
            priority = "Low"
        }
        time.Sleep(500 * time.Millisecond)
        fmt.Fprintf(w, "Hello from Server %d, serving req with priority: %s\n", port, priority)
    })

    srv := &http.Server{
        Addr:    ":" + strconv.Itoa(port),
        Handler: mux,
    }

    serversMu.Lock()
    servers[port] = srv
    serversMu.Unlock()

    log.Printf("Server running on :%d", port)
    if err := srv.ListenAndServe(); err != nil {
        log.Printf("Server %d stopped: %v", port, err)
    }
}

func StopServer(port int) {
    serversMu.Lock()
    srv, ok := servers[port]
    serversMu.Unlock()
    if !ok {
        log.Printf("No server running on port %d", port)
        return
    }

    // Gracefully shutdown with a timeout
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    if err := srv.Shutdown(ctx); err != nil {
        log.Printf("Server %d shutdown error: %v", port, err)
    } else {
        log.Printf("Server %d stopped", port)
    }

    // Remove from map
    delete(servers, port)
}