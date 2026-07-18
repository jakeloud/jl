package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jakeloud/jl/entities"
	"github.com/jakeloud/jl/logger"
	"github.com/jakeloud/jl/server"
	"github.com/jakeloud/jl/setup"
)

func main() {
	dry := flag.Bool("dry", false, "Run in dry mode")
	daemon := flag.Bool("d", false, "Run in daemon mode")
	flag.Parse()
	entities.SetDry(*dry)

	if !(*daemon) {
		setup.Start(*dry)
		return
	}

	entities.SetReleaseFailureNotifier(logger.Log)
	if err := server.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
	defer entities.ShutdownReleases(5 * time.Second)
	log.Println("Started server")

	httpServer := &http.Server{Addr: ":666"}
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- httpServer.ListenAndServe()
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signals)

	select {
	case err := <-serverErr:
		if err != http.ErrServerClosed {
			log.Printf("Server failed: %v", err)
		}
		if !entities.ShutdownReleases(5 * time.Second) {
			log.Println("Timed out waiting for releases to stop")
		}
	case <-signals:
		if !entities.ShutdownReleases(5 * time.Second) {
			log.Println("Timed out waiting for releases to stop")
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := httpServer.Shutdown(ctx); err != nil {
			log.Printf("Server shutdown failed: %v", err)
		}
		cancel()
	}
}
