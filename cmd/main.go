package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"net/http"

	"github.com/itzmanish/sipnexus"
	"github.com/itzmanish/sipnexus/pkg/logger"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	// Initialize logger
	log := logger.NewLogger()

	// Create a new SIP server
	sipServer, err := sipnexus.NewServer(log)
	if err != nil {
		log.Fatal("Failed to create SIP server: " + err.Error())
	}

	// Start Prometheus metrics server
	http.Handle("/metrics", promhttp.Handler())
	go http.ListenAndServe(":8080", nil)

	// Create a deadline to wait for
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the SIP server
	go func() {
		if err := sipServer.Start(ctx); err != nil {
			log.Fatal("Failed to start SIP server: " + err.Error())
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")
	cancel()

	// Doesn't block if no connections, but will otherwise wait
	// until the timeout deadline.
	if err := sipServer.Shutdown(); err != nil {
		log.Error("Server forced to shutdown: " + err.Error())
	}

	log.Info("Server exiting")
}
