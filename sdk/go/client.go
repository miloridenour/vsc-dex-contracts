package vscdex

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/hasura/go-graphql-client"
	vscnode "vsc-node/modules/state-processing"
	"vsc-node/modules/transaction-pool"
)

// Client provides SDK methods for VSC DEX mapping operations
type Client struct {
	config             Config
	transactionCreator *transactionpool.TransactionCrafter
}

type Config struct {
	Endpoint   string
	Username   string
	ActiveKey  string
	Contracts  ContractAddresses
}

type ContractAddresses struct {
	BtcMapping     string
	TokenRegistry  string
	DexRouter      string
}

// NewClient creates a new VSC DEX client
func NewClient(config Config) *Client {
	return &Client{
		config:             config,
		transactionCreator: nil, // TODO: Initialize TransactionCrafter if needed
	}
}

// RegisterMappedToken registers a new mapped token in the registry
func (c *Client) RegisterMappedToken(ctx context.Context, symbol string, decimals uint8, owner string) error {
	// Call token-registry contract
	payload := fmt.Sprintf(`{
		"contract": "%s",
		"method": "registerToken",
		"args": {
			"symbol": "%s",
			"decimals": %d,
			"owner": "%s"
		}
	}`, c.config.Contracts.TokenRegistry, symbol, decimals, owner)

	return c.broadcastTx(ctx, payload)
}

// SubmitBtcHeaders submits Bitcoin headers to the mapping contract
func (c *Client) SubmitBtcHeaders(ctx context.Context, headers []byte) error {
	// Call btc-mapping contract
	payload := fmt.Sprintf(`{
		"contract": "%s",
		"method": "submitHeaders",
		"args": {
			"headers": "%x"
		}
	}`, c.config.Contracts.BtcMapping, headers)

	return c.broadcastTx(ctx, payload)
}

// ProveBtcDeposit submits a Bitcoin deposit proof
func (c *Client) ProveBtcDeposit(ctx context.Context, proof []byte) (uint64, error) {
	// Call btc-mapping contract
	payload := fmt.Sprintf(`{
		"contract": "%s",
		"method": "proveDeposit",
		"args": {
			"proof": "%x"
		}
	}`, c.config.Contracts.BtcMapping, proof)

	err := c.broadcastTx(ctx, payload)
	if err != nil {
		return 0, err
	}

	// In a real implementation, we would parse the transaction response
	// to get the actual minted amount. For now, simulate based on proof.
	if len(proof) >= 44 {
		// Extract amount from proof (bytes 36-43)
		amount := uint64(proof[36]) | uint64(proof[37])<<8 | uint64(proof[38])<<16 | uint64(proof[39])<<24 |
		          uint64(proof[40])<<32 | uint64(proof[41])<<40 | uint64(proof[42])<<48 | uint64(proof[43])<<56
		return amount, nil
	}

	return 0, fmt.Errorf("invalid proof format")
}

// RequestBtcWithdrawal burns mapped BTC tokens for withdrawal
func (c *Client) RequestBtcWithdrawal(ctx context.Context, amount uint64, btcAddress string) error {
	payload := fmt.Sprintf(`{
		"contract": "%s",
		"method": "requestWithdraw",
		"args": {
			"amount": %d,
			"btcAddress": "%s"
		}
	}`, c.config.Contracts.BtcMapping, amount, btcAddress)

	return c.broadcastTx(ctx, payload)
}

// ComputeDexRoute computes an optimal DEX swap route
func (c *Client) ComputeDexRoute(ctx context.Context, fromAsset, toAsset string, amount int64) (*RouteResult, error) {
	// Call router service HTTP API
	routerURL := fmt.Sprintf("%s/router/route", c.config.Endpoint)

	payload := map[string]interface{}{
		"fromAsset": fromAsset,
		"toAsset":   toAsset,
		"amount":    amount,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", routerURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to query router for route: %v", err)
		return nil, fmt.Errorf("failed to compute route: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("router returned status %d", resp.StatusCode)
	}

	// Parse response
	var result RouteResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse route response: %w", err)
	}

	return &result, nil
}

type RouteResult struct {
	AmountOut  int64    `json:"amount_out"`
	Route      []string `json:"route"`
	PriceImpact float64  `json:"price_impact"`
	Fee         int64    `json:"fee"`
}

// ExecuteDexSwap executes a computed DEX swap
func (c *Client) ExecuteDexSwap(ctx context.Context, route *RouteResult) error {
	// Serialize the route to JSON
	routeJSON, err := json.Marshal(route.Route)
	if err != nil {
		return fmt.Errorf("failed to serialize route: %w", err)
	}

	// Call DEX contract with computed route
	payload := fmt.Sprintf(`{
		"contract": "%s",
		"method": "executeSwap",
		"args": {
			"route": %s,
			"amountOut": %d,
			"fee": %d
		}
	}`, c.config.Contracts.DexRouter, string(routeJSON), route.AmountOut, route.Fee)

	return c.broadcastTx(ctx, payload)
}

// ExecuteDexSwapRouter implements the router.DEXExecutor interface
// This allows the SDK client to be injected into the router service
func (c *Client) ExecuteDexSwapRouter(ctx context.Context, amountOut int64, route []string, fee int64) error {
	// Create SDK RouteResult from parameters
	sdkRoute := &RouteResult{
		AmountOut:  amountOut,
		Route:      route,
		PriceImpact: 0, // TODO: Calculate price impact
		Fee:        fee,
	}

	return c.ExecuteDexSwap(ctx, sdkRoute)
}

// broadcastTx broadcasts a transaction to VSC
func (c *Client) broadcastTx(ctx context.Context, payload string) error {
	// Parse the payload to extract contract call parameters
	var contractCall map[string]interface{}
	if err := json.Unmarshal([]byte(payload), &contractCall); err != nil {
		return fmt.Errorf("failed to parse contract call payload: %w", err)
	}

	contractID, _ := contractCall["contract"].(string)
	method, _ := contractCall["method"].(string)
	args, _ := contractCall["args"].(map[string]interface{})

	// Serialize args to JSON string (VscContractCall.Payload is string, not map)
	argsJSON, err := json.Marshal(args)
	if err != nil {
		return fmt.Errorf("failed to marshal contract call args: %w", err)
	}

	// Create VSC contract call transaction
	vscCall := &transactionpool.VscContractCall{
		Caller:     c.config.Username, // Use configured username as caller
		ContractId: contractID,
		RcLimit:    1000, // Default RC limit
		Action:     method,
		Payload:    string(argsJSON), // Payload must be JSON string
		NetId:      "vsc-mainnet",
	}

	// Serialize the contract call
	op, err := vscCall.SerializeVSC()
	if err != nil {
		return fmt.Errorf("failed to serialize contract call: %w", err)
	}

	// Create VSC transaction
	tx := transactionpool.VSCTransaction{
		Ops: []transactionpool.VSCTransactionOp{op},
		Nonce: 0, // TODO: Implement proper nonce management
	}

	// For now, create mock signed transaction
	// TODO: Implement proper transaction signing with active key
	mockTxBytes, _ := json.Marshal(tx)
	mockSigBytes := []byte("mock_signature_" + string(mockTxBytes))

	txStr := base64.StdEncoding.EncodeToString(mockTxBytes)
	sigStr := base64.StdEncoding.EncodeToString(mockSigBytes)

	// Create GraphQL client
	gqlClient := graphql.NewClient(c.config.Endpoint+"/api/v1/graphql", nil)

	// Execute GraphQL mutation
	var mutation struct {
		SubmitTransactionV1 struct {
			Id graphql.String `graphql:"id"`
		} `graphql:"submitTransactionV1(tx: $tx, sig: $sig)"`
	}

	err = gqlClient.Query(ctx, &mutation, map[string]interface{}{
		"tx":  graphql.String(txStr),
		"sig": graphql.String(sigStr),
	})

	if err != nil {
		log.Printf("Failed to broadcast transaction: %v", err)
		return fmt.Errorf("failed to broadcast transaction: %w", err)
	}

	log.Printf("Transaction broadcasted successfully, ID: %s", mutation.SubmitTransactionV1.Id)
	return nil
}

// GetPools queries available liquidity pools from indexer
func (c *Client) GetPools(ctx context.Context) ([]PoolInfo, error) {
	// Make HTTP call to indexer service
	indexerURL := fmt.Sprintf("%s/indexer/pools", c.config.Endpoint)

	req, err := http.NewRequestWithContext(ctx, "GET", indexerURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to query indexer for pools: %v", err)
		return nil, fmt.Errorf("failed to query pools: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("indexer returned status %d", resp.StatusCode)
	}

	// Parse response
	var pools []PoolInfo
	if err := json.NewDecoder(resp.Body).Decode(&pools); err != nil {
		return nil, fmt.Errorf("failed to parse pools response: %w", err)
	}

	return pools, nil
}

// GetTokens queries registered tokens from indexer
func (c *Client) GetTokens(ctx context.Context) ([]TokenInfo, error) {
	// Make HTTP call to indexer service
	indexerURL := fmt.Sprintf("%s/indexer/tokens", c.config.Endpoint)

	req, err := http.NewRequestWithContext(ctx, "GET", indexerURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to query indexer for tokens: %v", err)
		return nil, fmt.Errorf("failed to query tokens: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("indexer returned status %d", resp.StatusCode)
	}

	// Parse response
	var tokens []TokenInfo
	if err := json.NewDecoder(resp.Body).Decode(&tokens); err != nil {
		return nil, fmt.Errorf("failed to parse tokens response: %w", err)
	}

	return tokens, nil
}

type PoolInfo struct {
	ID       string  `json:"id"`
	Asset0   string  `json:"asset0"`
	Asset1   string  `json:"asset1"`
	Reserve0 uint64  `json:"reserve0"`
	Reserve1 uint64  `json:"reserve1"`
	Fee      float64 `json:"fee"`
}

type TokenInfo struct {
	Symbol      string `json:"symbol"`
	Decimals    uint8  `json:"decimals"`
	ContractID  string `json:"contract_id"`
	Description string `json:"description"`
}
