// this file contains the function for graceful shutdown
package utils

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
	"sync"

	"lb/models"
	"lb/server"
	"lb/config"
)


// GracefulShutdown listens for OS signals (Ctrl+C, SIGTERM) and gracefully shuts down servers.
func GracefulShutdown (lb *models.LoadBalancer, mainSrv *http.Server, wg *sync.WaitGroup ){
	defer wg.Done()
	stop := make(chan os.Signal, config.BufferSize)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// we should block until signal received
	<-stop
	log.Println("Shutting down Load Balancer...\n")

	// grace period
	ctx, cancel := context.WithTimeout(context.Background(), config.GracePeriod * time.Second)
	defer cancel()

	// stop LB gracefully
	if err := mainSrv.Shutdown(ctx); err != nil {
		log.Printf("Erorr shutting down the Load Balancer: %v\n", err)
	} else {
		log.Println("Load Balancer has been shut down\n")
	}

	stopped := 0
	// need to stop backend servers as well
	for _, backend := range lb.Backends {
		portNum, err := strconv.Atoi(backend.URL.Port())
		if err == nil{
			server.StopServer(portNum)
			stopped++
		}
	}

	log.Printf("A total of %d backend servers stopped successfully\n", stopped)
}