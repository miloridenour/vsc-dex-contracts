package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/vsc-eco/vsc-dex-mapping/services/indexer"
)

func main() {
	var (
		httpEndpoint = flag.String("http-endpoint", "http://localhost:4000", "VSC GraphQL HTTP endpoint")
		wsEndpoint   = flag.String("ws-endpoint", "", "VSC GraphQL WebSocket endpoint (optional, will try WebSocket first if provided)")
		httpPort     = flag.String("http-port", "8081", "HTTP server port")
		contracts    = flag.String("contracts", "", "Comma-separated list of contract IDs to monitor")
	)
	flag.Parse()

	svc := indexer.NewService(*httpEndpoint, *httpPort)

	// Set WebSocket URL if provided (will attempt WebSocket first, fallback to polling)
	if *wsEndpoint != "" {
		svc.SetWebSocketURL(*wsEndpoint)
	}

	// Parse and set contracts to monitor
	if *contracts != "" {
		contractList := []string{}
		// Simple parsing - split by comma
		for _, c := range strings.Split(*contracts, ",") {
			c = strings.TrimSpace(c)
			if c != "" {
				contractList = append(contractList, c)
			}
		}
		svc.SetContracts(contractList)
	}

	// Handle graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if *wsEndpoint != "" {
			log.Printf("Starting indexer service on HTTP port %s, attempting WebSocket: %s (fallback: polling from %s)...", *httpPort, *wsEndpoint, *httpEndpoint)
		} else {
			log.Printf("Starting indexer service on HTTP port %s, polling from %s...", *httpPort, *httpEndpoint)
		}
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
