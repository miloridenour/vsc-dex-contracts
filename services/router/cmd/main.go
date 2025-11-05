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

	"github.com/vsc-eco/vsc-dex-mapping/services/router"
)

func main() {
	var (
		vscNode     = flag.String("vsc-node", "http://localhost:4000", "VSC node GraphQL endpoint")
		vscKey      = flag.String("vsc-key", "", "VSC active key for transactions")
		vscUsername = flag.String("vsc-username", "", "VSC username")
		port        = flag.String("port", "8080", "HTTP server port")
	)
	flag.Parse()

	config := router.VSCConfig{
		Endpoint: *vscNode,
		Key:      *vscKey,
		Username: *vscUsername,
	}

	svc := router.NewService(config)
	server := router.NewServer(svc, *port)

	// Handle graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("Starting router service on port %s...", *port)
		if err := server.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Server failed to start:", err)
		}
	}()

	<-c
	log.Println("Shutting down router service...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Stop(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Router service stopped")
}
