// this file can run with go run server.go, which will start ${num} servers based on the numServers variable

package server

import (
    "fmt"
    "log"
    "net/http"
    "strconv"
    "time"
    "context"
)

var servers = make(map[int]*http.Server)

func StartServer(port int) {
	mux := http.NewServeMux()
    mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintf(w, "Hello from Server %d\n", port)
    })

    srv := &http.Server{
        Addr:    ":" + strconv.Itoa(port),
        Handler: mux,
    }

    servers[port] = srv

    log.Printf("Server running on :%d", port)
    if err := srv.ListenAndServe(); err != nil {
        log.Printf("Server %d stopped: %v", port, err)
    }
}

func StopServer(port int) {
    srv, ok := servers[port]
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