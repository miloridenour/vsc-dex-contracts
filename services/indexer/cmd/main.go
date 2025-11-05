package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/vsc-eco/vsc-dex-mapping/services/indexer"
)

func main() {
	var (
		wsEndpoint = flag.String("ws-endpoint", "ws://localhost:4000/graphql", "VSC GraphQL WebSocket endpoint")
		httpPort   = flag.String("http-port", "8081", "HTTP server port")
	)
	flag.Parse()

	svc := indexer.NewService(*wsEndpoint, *httpPort)

	// Handle graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		log.Printf("Starting indexer service on HTTP port %s, WS: %s...", *httpPort, *wsEndpoint)
		if err := svc.Start(ctx); err != nil {
			log.Fatal("Indexer failed to start:", err)
		}
	}()

	<-c
	log.Println("Shutting down indexer service...")

	cancel()
	time.Sleep(2 * time.Second) // Give services time to shutdown

	log.Println("Indexer service stopped")
}
