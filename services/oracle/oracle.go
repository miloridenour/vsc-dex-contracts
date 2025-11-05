package oracle

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"log"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/wire"
	"github.com/vsc-eco/hivego"
)

// Service handles Bitcoin header submission to VSC btc-mapping contract
type Service struct {
	btcClient *rpcclient.Client
	vscClient *hivego.HiveRpc
	vscConfig VSCConfig
}

type VSCConfig struct {
	Endpoint string
	Key      string
	Username string
}

// NewService creates a new oracle service
func NewService(btcConfig *rpcclient.ConnConfig, vscConfig VSCConfig) (*Service, error) {
	btcClient, err := rpcclient.New(btcConfig, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create BTC client: %w", err)
	}

	vscClient := hivego.NewHiveRpc(vscConfig.Endpoint)

	return &Service{
		btcClient: btcClient,
		vscClient: vscClient,
		vscConfig: vscConfig,
	}, nil
}

// SubmitHeaders fetches new Bitcoin headers and submits them to the VSC contract
func (s *Service) SubmitHeaders(ctx context.Context) error {
	// Get latest block count
	latestHeight, err := s.btcClient.GetBlockCount()
	if err != nil {
		return fmt.Errorf("failed to get block count: %w", err)
	}

	// Get current contract tip from VSC
	contractTip := s.getContractTip(ctx) // TODO: implement

	// Submit headers from contractTip+1 to latestHeight-6 (confirmations)
	startHeight := contractTip + 1
	endHeight := latestHeight - 6 // Require 6 confirmations

	if startHeight > endHeight {
		log.Printf("No new headers to submit (start: %d, end: %d)", startHeight, endHeight)
		return nil
	}

	headers, err := s.fetchHeaders(startHeight, endHeight)
	if err != nil {
		return fmt.Errorf("failed to fetch headers: %w", err)
	}

	// Submit to contract
	return s.submitHeadersToContract(ctx, headers)
}

// fetchHeaders retrieves block headers from Bitcoin node
func (s *Service) fetchHeaders(startHeight, endHeight int64) ([]*wire.BlockHeader, error) {
	headers := make([]*wire.BlockHeader, 0, endHeight-startHeight+1)

	for height := startHeight; height <= endHeight; height++ {
		hash, err := s.btcClient.GetBlockHash(height)
		if err != nil {
			return nil, fmt.Errorf("failed to get block hash at height %d: %w", height, err)
		}

		header, err := s.btcClient.GetBlockHeader(hash)
		if err != nil {
			return nil, fmt.Errorf("failed to get block header for hash %s: %w", hash.String(), err)
		}

		headers = append(headers, header)
	}

	return headers, nil
}

// submitHeadersToContract submits headers to the VSC btc-mapping contract
func (s *Service) submitHeadersToContract(ctx context.Context, headers []*wire.BlockHeader) error {
	// Serialize headers
	var buf bytes.Buffer
	for _, header := range headers {
		if err := header.Serialize(&buf); err != nil {
			return fmt.Errorf("failed to serialize header: %w", err)
		}
	}

	headerBytes := buf.Bytes()

	// Call contract via VSC GraphQL/broadcast
	// TODO: Implement contract call
	log.Printf("Would submit %d headers (%d bytes) to contract", len(headers), len(headerBytes))

	return nil
}

// getContractTip retrieves the current tip height from the btc-mapping contract
func (s *Service) getContractTip(ctx context.Context) int64 {
	// TODO: Query contract state via GraphQL
	// For now, return a placeholder
	return 0
}

// VerifyDepositProof verifies a Bitcoin deposit proof and submits to contract
func (s *Service) VerifyDepositProof(ctx context.Context, proof []byte) error {
	// TODO: Parse SPV proof
	// TODO: Verify against local BTC headers
	// TODO: Submit to contract if valid

	return nil
}

// Close shuts down the service
func (s *Service) Close() {
	s.btcClient.Shutdown()
}
