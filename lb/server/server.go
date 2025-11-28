// this file defines the functions for the servers

package server

import (
    "log"
    "net/http"
    "strconv"
    "time"
    "context"
    "sync"

    "lb/config"
)

var (
    servers   = make(map[int]*http.Server)
    serversMu sync.Mutex
)

func StartServer(port int) {
	mux := http.NewServeMux()
    mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        time.Sleep(config.SleepTimeHandling * time.Millisecond)
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
        log.Printf("Server %d stopped\n", port)
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
    ctx, cancel := context.WithTimeout(context.Background(), config.SleepTimeOut * time.Second)
    defer cancel()

    if err := srv.Shutdown(ctx); err != nil {
        log.Printf("Server %d shutdown error: %v", port, err)
    } else {
        log.Printf("Server %d stopped", port)
    }

    // Remove from map
    delete(servers, port)
}