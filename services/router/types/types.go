package types

import (
	"context"
)

// MappingAdapter defines the interface for cross-chain asset mapping
type MappingAdapter interface {
	// Chain returns the blockchain symbol (BTC, ETH, SOL)
	Chain() string

	// GetMappedToken returns the VSC token symbol for this chain's native asset
	GetMappedToken() string

	// ValidateDepositProof validates a deposit proof for this chain
	ValidateDepositProof(ctx context.Context, proof []byte) (*DepositValidation, error)

	// CreateDepositProof creates a deposit proof from chain transaction data
	CreateDepositProof(ctx context.Context, txHash string, vout uint32) ([]byte, error)

	// GetRequiredConfirmations returns the number of confirmations required for deposits
	GetRequiredConfirmations() uint32

	// FormatAddress formats an address for this chain
	FormatAddress(address string) (string, error)

	// GetContractAddress returns the mapping contract address for this chain
	GetContractAddress() string
}

// DepositValidation represents the result of validating a deposit proof
type DepositValidation struct {
	Valid      bool
	Amount     uint64
	Recipient  string
	TxHash     string
	BlockHeight uint64
	Error      string
}

// Errors

type ErrUnsupportedChain struct {
	Chain string
}

func (e ErrUnsupportedChain) Error() string {
	return "unsupported chain: " + e.Chain
}

type ErrUnsupportedRoute struct {
	FromChain string
	ToChain   string
}

func (e ErrUnsupportedRoute) Error() string {
	return "unsupported cross-chain route: " + e.FromChain + " -> " + e.ToChain
}



