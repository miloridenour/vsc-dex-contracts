package main

import (
	"context"
	"flag"
	"log"
	"time"

	"github.com/btcsuite/btcd/rpcclient"
	"github.com/vsc-eco/vsc-dex-mapping/services/oracle"
)

func main() {
	var (
		btcHost     = flag.String("btc-host", "localhost:8332", "Bitcoin RPC host:port")
		btcUser     = flag.String("btc-user", "vsc-user", "Bitcoin RPC username")
		btcPass     = flag.String("btc-pass", "vsc-pass", "Bitcoin RPC password")
		vscNode     = flag.String("vsc-node", "http://localhost:4000", "VSC node GraphQL endpoint")
		vscKey      = flag.String("vsc-key", "", "VSC active key for contract calls")
		vscUsername = flag.String("vsc-username", "", "VSC username")
		interval    = flag.Duration("interval", 10*time.Minute, "Header submission interval")
	)
	flag.Parse()

	// Bitcoin RPC client config
	btcConfig := &rpcclient.ConnConfig{
		Host:         *btcHost,
		User:         *btcUser,
		Pass:         *btcPass,
		HTTPPostMode: true,
		DisableTLS:   true,
	}

	// VSC client config
	vscConfig := oracle.VSCConfig{
		Endpoint: *vscNode,
		Key:      *vscKey,
		Username: *vscUsername,
	}

	// Create oracle service
	svc, err := oracle.NewService(btcConfig, vscConfig)
	if err != nil {
		log.Fatal("Failed to create oracle service:", err)
	}

	ctx := context.Background()

	// Start header submission loop
	log.Println("Starting BTC oracle service...")
	ticker := time.NewTicker(*interval)
	defer ticker.Stop()

	// Initial submission
	if err := svc.SubmitHeaders(ctx); err != nil {
		log.Printf("Initial header submission failed: %v", err)
	}

	for {
		select {
		case <-ticker.C:
			if err := svc.SubmitHeaders(ctx); err != nil {
				log.Printf("Header submission failed: %v", err)
			}
		case <-ctx.Done():
			log.Println("Shutting down oracle service...")
			return
		}
	}
}
