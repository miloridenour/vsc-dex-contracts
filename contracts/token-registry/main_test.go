package main

import (
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"
)

// Mock SDK for testing token registry
type tokenMockSDK struct {
	state map[string]interface{}
}

var tokenTestSDK = &tokenMockSDK{state: make(map[string]interface{})}

func (m *tokenMockSDK) SetState(key string, value interface{}) {
	m.state[key] = value
}

func (m *tokenMockSDK) GetState(key string) interface{} {
	return m.state[key]
}

func (m *tokenMockSDK) GetCaller() string {
	return "registry-owner"
}

func (m *tokenMockSDK) GetBlockTime() uint64 {
	return 1234567890
}

func TestTokenRegistryInitialization(t *testing.T) {
	// Test contract initialization
	init()

	state := tokenTestSDK.GetState("state").(ContractState)

	assert.NotNil(t, state.Tokens)
	assert.Equal(t, "registry-owner", state.Owner)
}

func TestRegisterToken(t *testing.T) {
	// Initialize contract
	init()

	symbol := "TEST"
	decimals := uint8(18)
	contractID := "test-contract-id"

	result := registerToken(
		unsafe.Pointer(&([]byte(symbol))[0]), uint32(len(symbol)),
		decimals,
		unsafe.Pointer(&([]byte(contractID))[0]), uint32(len(contractID)),
	)

	assert.Equal(t, uint32(0), result) // success
}

func TestGetToken(t *testing.T) {
	// Initialize and register token
	init()
	symbol := "TEST"
	decimals := uint8(18)
	contractID := "test-contract-id"

	registerToken(
		unsafe.Pointer(&([]byte(symbol))[0]), uint32(len(symbol)),
		decimals,
		unsafe.Pointer(&([]byte(contractID))[0]), uint32(len(contractID)),
	)

	// Get token info
	result := getToken(unsafe.Pointer(&([]byte(symbol))[0]), uint32(len(symbol)))
	assert.NotNil(t, result) // should return token data
}

func TestUpdateSupply(t *testing.T) {
	// Initialize and register token
	init()
	symbol := "TEST"
	decimals := uint8(18)
	contractID := "test-contract-id"

	registerToken(
		unsafe.Pointer(&([]byte(symbol))[0]), uint32(len(symbol)),
		decimals,
		unsafe.Pointer(&([]byte(contractID))[0]), uint32(len(contractID)),
	)

	// Update supply
	newSupply := uint64(1000000)
	result := updateSupply(
		unsafe.Pointer(&([]byte(symbol))[0]), uint32(len(symbol)),
		newSupply,
	)

	assert.Equal(t, uint32(0), result) // success
}

func TestGetAllTokens(t *testing.T) {
	// Initialize contract
	init()

	// Register multiple tokens
	tokens := []struct {
		symbol  string
		decimals uint8
		contract string
	}{
		{"BTC", 8, "btc-mapping"},
		{"ETH", 18, "eth-mapping"},
		{"SOL", 9, "sol-mapping"},
	}

	for _, token := range tokens {
		registerToken(
			unsafe.Pointer(&([]byte(token.symbol))[0]), uint32(len(token.symbol)),
			token.decimals,
			unsafe.Pointer(&([]byte(token.contract))[0]), uint32(len(token.contract)),
		)
	}

	// Get all tokens
	result := getAllTokens()
	assert.NotNil(t, result) // should return token array
}
