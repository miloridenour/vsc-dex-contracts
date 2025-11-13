package main

import (
	"unsafe"

	"github.com/vsc-eco/go-vsc-node/sdk"
)

// Token Registry Contract
// Manages registration of wrapped/mapped assets for DEX integration

type TokenInfo struct {
	Symbol    string
	Decimals  uint8
	Owner     string // contract address that controls supply
	TotalSupply uint64
	CreatedAt   uint64
}

type ContractState struct {
	Tokens map[string]TokenInfo // symbol -> info
	Owner  string              // registry owner
}

// Initialize contract
//export init
func init() {
	state := ContractState{
		Tokens: make(map[string]TokenInfo),
		Owner:  "", // Set during deployment
	}

	sdk.SetState("state", state)
}

// Register a new token
//export registerToken
func registerToken(symbolPtr unsafe.Pointer, symbolLen uint32, decimals uint8, ownerPtr unsafe.Pointer, ownerLen uint32) uint32 {
	caller := sdk.GetCaller()
	state := sdk.GetState("state").(ContractState)

	// Only registry owner can register tokens
	if caller != state.Owner {
		return 1 // unauthorized
	}

	symbol := string(unsafe.Slice((*byte)(symbolPtr), symbolLen))
	owner := string(unsafe.Slice((*byte)(ownerPtr), ownerLen))

	token := TokenInfo{
		Symbol:    symbol,
		Decimals:  decimals,
		Owner:     owner,
		TotalSupply: 0,
		CreatedAt: sdk.GetBlockTime(),
	}

	state.Tokens[symbol] = token
	sdk.SetState("state", state)

	// Emit TokenRegistered event
	// TODO: emit event

	return 0 // success
}

// Get token info
//export getToken
func getToken(symbolPtr unsafe.Pointer, symbolLen uint32) unsafe.Pointer {
	symbol := string(unsafe.Slice((*byte)(symbolPtr), symbolLen))
	state := sdk.GetState("state").(ContractState)

	token, exists := state.Tokens[symbol]
	if !exists {
		return nil
	}

	// Return token info as JSON
	// TODO: serialize to JSON bytes

	return nil
}

// Update token supply (only token owner can call)
//export updateSupply
func updateSupply(symbolPtr unsafe.Pointer, symbolLen uint32, newSupply uint64) uint32 {
	caller := sdk.GetCaller()
	symbol := string(unsafe.Slice((*byte)(symbolPtr), symbolLen))

	state := sdk.GetState("state").(ContractState)
	token, exists := state.Tokens[symbol]
	if !exists {
		return 1 // token not found
	}

	// Only token owner can update supply
	if caller != token.Owner {
		return 2 // unauthorized
	}

	token.TotalSupply = newSupply
	state.Tokens[symbol] = token
	sdk.SetState("state", state)

	return 0 // success
}

// Get all registered tokens
//export getAllTokens
func getAllTokens() unsafe.Pointer {
	state := sdk.GetState("state").(ContractState)

	// Return array of tokens as JSON
	// TODO: serialize to JSON bytes

	return nil
}

func main() {
	// Contract entry point
}


