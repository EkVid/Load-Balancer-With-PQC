// this file can run with go run server.go, which will start ${num} servers based on the numServers variable

package main

import (
    "fmt"
    "log"
    "net/http"
    "strconv"
    "time"
)


func startServer(port int) {
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

func main(){
	numServers := 5
	startingPort := 8081

	for i := 0; i < numServers; i++ {
		port := startingPort + i
		go startServer(port)
		time.Sleep(200 * time.Millisecond) // avoid race
	}

	// keeps main alive forever
	select{}
}