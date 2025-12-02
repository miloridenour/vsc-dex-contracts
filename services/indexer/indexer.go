package indexer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Service indexes VSC DEX and bridge events into read models
type Service struct {
	httpURL      string
	wsURL        string // Optional WebSocket URL for when VSC supports subscriptions
	readers      []ReadModel
	mu           sync.RWMutex
	server       *Server
	conn         *websocket.Conn // WebSocket connection (if using subscriptions)
	lastBlock    uint64
	pollInterval time.Duration
	contracts    []string // Contract IDs to monitor
	useWebSocket bool     // Whether to attempt WebSocket subscriptions first
}

type ReadModel interface {
	HandleEvent(event VSCEvent) error
	QueryPools() ([]PoolInfo, error)
}

// PoolInfo represents indexed pool data
type PoolInfo struct {
	ID          string  `json:"id"`
	Asset0      string  `json:"asset0"`
	Asset1      string  `json:"asset1"`
	Reserve0    uint64  `json:"reserve0"`
	Reserve1    uint64  `json:"reserve1"`
	Fee         float64 `json:"fee"`
	TotalSupply uint64  `json:"total_supply"`
}



// VSCEvent represents a VSC blockchain event
type VSCEvent struct {
	Type        string          `json:"type"`
	Contract    string          `json:"contract"`
	Method      string          `json:"method"`
	Args        json.RawMessage `json:"args"`
	BlockHeight uint64          `json:"block_height"`
	TxID        string          `json:"tx_id"`
}

// NewService creates a new indexer service
func NewService(httpURL string, port string) *Service {
	svc := &Service{
		httpURL:      httpURL,
		wsURL:        "", // Will be set if WebSocket endpoint provided
		readers:      make([]ReadModel, 0),
		pollInterval: 5 * time.Second, // Poll every 5 seconds
		contracts:    []string{},      // Will be set via SetContracts
		useWebSocket: false,           // Default to polling
	}

	// Add default DEX read model
	dexReader := NewDexReadModel()
	svc.AddReader(dexReader)

	// Create HTTP server
	svc.server = NewServer(svc, port)

	return svc
}

// SetWebSocketURL sets the WebSocket endpoint and enables WebSocket mode
func (s *Service) SetWebSocketURL(wsURL string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.wsURL = wsURL
	s.useWebSocket = wsURL != ""
}

// SetContracts sets the contract IDs to monitor
func (s *Service) SetContracts(contracts []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.contracts = contracts
}

// AddReader adds a read model to the indexer
func (s *Service) AddReader(reader ReadModel) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.readers = append(s.readers, reader)
}

// Start begins indexing events and starts HTTP server
func (s *Service) Start(ctx context.Context) error {
	// Start HTTP server in background
	go func() {
		if err := s.server.Start(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	// Try WebSocket first if enabled, fallback to polling
	s.mu.RLock()
	useWS := s.useWebSocket
	wsURL := s.wsURL
	s.mu.RUnlock()

	if useWS && wsURL != "" {
		log.Printf("Attempting WebSocket subscription to %s...", wsURL)
		if err := s.startWebSocketIndexing(ctx); err != nil {
			log.Printf("WebSocket indexing failed (will fallback to polling): %v", err)
			// Fall through to polling
		} else {
			return nil // WebSocket succeeded
		}
	}

	// Use polling-based indexing (default or fallback)
	log.Printf("Using polling-based indexing (WebSocket not available or failed)")
	return s.startPolling(ctx)
}

// startPolling begins polling VSC GraphQL for new transactions and events
func (s *Service) startPolling(ctx context.Context) error {
	log.Printf("Starting polling-based indexing from %s (interval: %v)", s.httpURL, s.pollInterval)

	// Get initial block height
	if err := s.updateLastBlock(ctx); err != nil {
		log.Printf("Warning: Failed to get initial block height: %v", err)
	}

	ticker := time.NewTicker(s.pollInterval)
	defer ticker.Stop()

	// Poll immediately, then on interval
	if err := s.pollForEvents(ctx); err != nil {
		log.Printf("Error in initial poll: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := s.pollForEvents(ctx); err != nil {
				log.Printf("Error polling for events: %v", err)
				// Continue polling even on errors
			}
		}
	}
}

// updateLastBlock gets the current block height from VSC
func (s *Service) updateLastBlock(ctx context.Context) error {
	query := `query {
		localNodeInfo {
			last_processed_block
		}
	}`

	var result struct {
		Data struct {
			LocalNodeInfo struct {
				LastProcessedBlock uint64 `json:"last_processed_block"`
			} `json:"localNodeInfo"`
		} `json:"data"`
		Errors []map[string]interface{} `json:"errors,omitempty"`
	}

	if err := s.executeGraphQLQuery(ctx, query, nil, &result); err != nil {
		return err
	}

	if len(result.Errors) > 0 {
		return fmt.Errorf("GraphQL errors: %v", result.Errors)
	}

	s.mu.Lock()
	s.lastBlock = result.Data.LocalNodeInfo.LastProcessedBlock
	s.mu.Unlock()

	return nil
}

// pollForEvents polls for new transactions and contract outputs
func (s *Service) pollForEvents(ctx context.Context) error {
	s.mu.RLock()
	lastBlock := s.lastBlock
	contracts := s.contracts
	s.mu.RUnlock()

	// Poll for contract outputs (which contain event information)
	for _, contractID := range contracts {
		if err := s.pollContractOutputs(ctx, contractID, lastBlock); err != nil {
			log.Printf("Error polling contract outputs for %s: %v", contractID, err)
		}
	}

	// Update last block height
	return s.updateLastBlock(ctx)
}

// pollContractOutputs polls for contract outputs from a specific contract
func (s *Service) pollContractOutputs(ctx context.Context, contractID string, fromBlock uint64) error {
	query := `query FindContractOutput($filter: ContractOutputFilter!) {
		findContractOutput(filterOptions: $filter) {
			id
			block_height
			timestamp
			contract_id
			inputs
			state_merkle
			results {
				ret
				ok
			}
		}
	}`

	variables := map[string]interface{}{
		"filter": map[string]interface{}{
			"byContract": contractID,
			"limit":      100, // Get up to 100 recent outputs
		},
	}

	var result struct {
		Data struct {
			FindContractOutput []struct {
				ID          string   `json:"id"`
				BlockHeight int64    `json:"block_height"`
				Timestamp   string   `json:"timestamp"`
				ContractID  string   `json:"contract_id"`
				Inputs      []string `json:"inputs"`
				Results     []struct {
					Ret string `json:"ret"`
					Ok  bool   `json:"ok"`
				} `json:"results"`
			} `json:"findContractOutput"`
		} `json:"data"`
		Errors []map[string]interface{} `json:"errors,omitempty"`
	}

	if err := s.executeGraphQLQuery(ctx, query, variables, &result); err != nil {
		return err
	}

	if len(result.Errors) > 0 {
		return fmt.Errorf("GraphQL errors: %v", result.Errors)
	}

	// Process contract outputs and extract events
	for _, output := range result.Data.FindContractOutput {
		if int64(fromBlock) < output.BlockHeight {
			// This is a new output, process it
			// Contract outputs contain the result of contract calls, which we can parse as events
			// The actual event parsing depends on the contract's event structure
			// For now, we'll create a generic event from the contract output
			event := VSCEvent{
				Type:        "contract_output",
				Contract:    output.ContractID,
				Method:      "", // Will be extracted from result if available
				BlockHeight: uint64(output.BlockHeight),
				TxID:        output.ID,
				Args:        json.RawMessage("{}"), // Contract output data would go here
			}

			// Try to extract method/event info from results
			if len(output.Results) > 0 && output.Results[0].Ret != "" {
				// The ret field may contain JSON with event information
				// This is contract-specific and would need to be parsed based on contract structure
				event.Args = json.RawMessage(output.Results[0].Ret)
			}

			s.handleEvent(event)
		}
	}

	return nil
}

// executeGraphQLQuery executes a GraphQL query via HTTP POST
func (s *Service) executeGraphQLQuery(ctx context.Context, query string, variables map[string]interface{}, result interface{}) error {
	payload := map[string]interface{}{
		"query": query,
	}
	if variables != nil {
		payload["variables"] = variables
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal GraphQL request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.httpURL+"/api/v1/graphql", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute GraphQL query: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GraphQL request failed with status %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("failed to decode GraphQL response: %w", err)
	}

	return nil
}

// startWebSocketIndexing attempts to use WebSocket subscriptions (for when VSC supports them)
func (s *Service) startWebSocketIndexing(ctx context.Context) error {
	// Use graphql-ws subprotocol for WebSocket connection
	dialer := websocket.Dialer{
		Subprotocols: []string{"graphql-ws"},
	}

	s.mu.RLock()
	wsURL := s.wsURL
	s.mu.RUnlock()

	conn, resp, err := dialer.Dial(wsURL, nil)
	if err != nil {
		if resp != nil {
			return fmt.Errorf("WebSocket handshake failed with status %d: %w", resp.StatusCode, err)
		}
		return fmt.Errorf("WebSocket connection failed: %w", err)
	}

	s.mu.Lock()
	s.conn = conn
	s.mu.Unlock()

	defer conn.Close()

	// VSC GraphQL uses graphql-ws protocol
	// First, send connection_init message
	initMsg := map[string]interface{}{
		"type": "connection_init",
	}
	if err := conn.WriteJSON(initMsg); err != nil {
		return fmt.Errorf("failed to send connection_init: %w", err)
	}

	// Wait for connection_ack
	var ackMsg map[string]interface{}
	if err := conn.ReadJSON(&ackMsg); err != nil {
		return fmt.Errorf("failed to receive connection_ack: %w", err)
	}
	if ackMsg["type"] != "connection_ack" {
		return fmt.Errorf("expected connection_ack, got: %v", ackMsg)
	}

	// Subscribe to DEX and bridge events using graphql-ws protocol
	s.mu.RLock()
	contracts := s.contracts
	s.mu.RUnlock()

	// Build contract filter
	contractFilter := `["dex-router"]`
	if len(contracts) > 0 {
		contractList := "["
		for i, c := range contracts {
			if i > 0 {
				contractList += ", "
			}
			contractList += fmt.Sprintf(`"%s"`, c)
		}
		contractList += "]"
		contractFilter = contractList
	}

	subscription := map[string]interface{}{
		"id":   "1",
		"type": "start",
		"payload": map[string]interface{}{
			"query": fmt.Sprintf(`subscription {
				events(filter: {contracts: %s}) {
					type
					contract
					method
					args
					blockHeight
					txId
				}
			}`, contractFilter),
		},
	}

	if err := conn.WriteJSON(subscription); err != nil {
		return fmt.Errorf("failed to send subscription: %w", err)
	}

	log.Printf("WebSocket subscription established successfully")

	// Handle incoming events
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			var response struct {
				ID      string                 `json:"id"`
				Type    string                 `json:"type"`
				Payload map[string]interface{} `json:"payload"`
			}

			err := conn.ReadJSON(&response)
			if err != nil {
				return fmt.Errorf("WebSocket read error: %w", err)
			}

			// Handle different message types
			switch response.Type {
			case "data":
				// Parse the event from payload
				if payloadData, ok := response.Payload["data"].(map[string]interface{}); ok {
					if events, ok := payloadData["events"].([]interface{}); ok {
						for _, eventData := range events {
							if eventMap, ok := eventData.(map[string]interface{}); ok {
								event := parseVSCEvent(eventMap)
								if event != nil {
									s.handleEvent(*event)
								}
							}
						}
					}
				}
			case "error":
				log.Printf("WebSocket subscription error: %v", response.Payload)
				return fmt.Errorf("subscription error: %v", response.Payload)
			case "complete":
				log.Printf("WebSocket subscription completed")
				return nil
			}
		}
	}
}

// parseVSCEvent parses a VSC event from a map (for WebSocket subscriptions)
func parseVSCEvent(eventMap map[string]interface{}) *VSCEvent {
	event := &VSCEvent{}

	if v, ok := eventMap["type"].(string); ok {
		event.Type = v
	}
	if v, ok := eventMap["contract"].(string); ok {
		event.Contract = v
	}
	if v, ok := eventMap["method"].(string); ok {
		event.Method = v
	}
	if v, ok := eventMap["txId"].(string); ok {
		event.TxID = v
	}
	if v, ok := eventMap["blockHeight"].(float64); ok {
		event.BlockHeight = uint64(v)
	}
	if args, ok := eventMap["args"].(map[string]interface{}); ok {
		argsJSON, err := json.Marshal(args)
		if err == nil {
			event.Args = argsJSON
		}
	}

	return event
}

// handleEvent processes an incoming VSC event
func (s *Service) handleEvent(event VSCEvent) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, reader := range s.readers {
		if err := reader.HandleEvent(event); err != nil {
			log.Printf("Error handling event in reader: %v", err)
		}
	}
}

// QueryPools returns all indexed pools
func (s *Service) QueryPools() ([]PoolInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var allPools []PoolInfo
	for _, reader := range s.readers {
		pools, err := reader.QueryPools()
		if err != nil {
			continue // Skip readers that don't support this query
		}
		allPools = append(allPools, pools...)
	}

	return allPools, nil
}


