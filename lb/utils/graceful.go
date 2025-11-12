// this file contains the function for graceful shutdown
package utils

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
	"sync"

	"lb/models"
	"lb/server"
)


// GracefulShutdown listens for OS signals (Ctrl+C, SIGTERM) and gracefully shuts down servers.
func GracefulShutdown (lb *models.LoadBalancer, mainSrv *http.Server, wg *sync.WaitGroup ){
	defer wg.Done()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// we should block until signal received
	<-stop
	fmt.Println("Shutting down Load Balancer...\n")

	// 10 seconds grace period
	ctx, cancel := context.WithTimeout(context.Background(), 10 * time.Second)
	defer cancel()

	// stop LB gracefully
	if err := mainSrv.Shutdown(ctx); err != nil {
		fmt.Printf("Erorr shutting down the Load Balancer: %v\n", err)
	} else {
		fmt.Println("Load Balancer has been shut down\n")
	}

	// need to stop backend servers as well
	for _, backend := range lb.Backends {
		portNum, err := strconv.Atoi(backend.URL.Port())
		if err == nil{
			server.StopServer(portNum)
		}
	}

	fmt.Println("All backend servers stopped successfully\n")
}