// this file can run with go run server.go, which will start ${num} servers based on the numServers variable

package server

import (
    "fmt"
    "log"
    "net/http"
    "strconv"
)


func StartServer(port int) {
	mux := http.NewServeMux()
    mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintf(w, "Hello from Server %d\n", port)
    })

    srv := &http.Server{
        Addr:    ":" + strconv.Itoa(port),
        Handler: mux,
    }

    log.Printf("Server running on :%d", port)
    if err := srv.ListenAndServe(); err != nil {
        log.Printf("Server %d stopped: %v", port, err)
    }
}