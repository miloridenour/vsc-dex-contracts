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

	vscdex "github.com/vsc-eco/vsc-dex-mapping/sdk/go"
	"github.com/vsc-eco/vsc-dex-mapping/services/router"
)

func main() {
	var (
		vscNode        = flag.String("vsc-node", "http://localhost:4000", "VSC node GraphQL endpoint")
		vscKey         = flag.String("vsc-key", "", "VSC active key for transactions")
		vscUsername    = flag.String("vsc-username", "", "VSC username")
		port           = flag.String("port", "8080", "HTTP server port")
		indexerEndpoint = flag.String("indexer-endpoint", "http://localhost:8081", "Indexer service HTTP endpoint")
		dexRouter      = flag.String("dex-router-contract", "", "DEX router contract ID")
	)
	flag.Parse()

	config := router.VSCConfig{
		Endpoint:          *vscNode,
		Key:               *vscKey,
		Username:          *vscUsername,
		DexRouterContract: *dexRouter,
	}

	// Create SDK client to use as DEXExecutor
	sdkClient := vscdex.NewClient(vscdex.Config{
		Endpoint: *vscNode,
		Username: *vscUsername,
		ActiveKey: *vscKey,
		Contracts: vscdex.ContractAddresses{
			DexRouter: *dexRouter,
		},
	})

	svc := router.NewService(config, sdkClient)
	
	// Connect router to indexer for real-time pool data
	if *indexerEndpoint != "" {
		poolQuerier := router.NewIndexerPoolQuerier(*indexerEndpoint)
		svc.SetPoolQuerier(poolQuerier)
		log.Printf("Router connected to indexer at %s", *indexerEndpoint)
	} else {
		log.Printf("Warning: No indexer endpoint provided, router will use hardcoded fallback pools")
	}
	
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
