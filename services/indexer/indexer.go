package indexer

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

// Service indexes VSC DEX and bridge events into read models
type Service struct {
	wsURL    string
	conn     *websocket.Conn
	readers  []ReadModel
	mu       sync.RWMutex
	server   *Server
}

type ReadModel interface {
	HandleEvent(event VSCEvent) error
	QueryPools() ([]PoolInfo, error)
	QueryTokens() ([]TokenInfo, error)
	QueryDeposits() ([]DepositInfo, error)
}

// PoolInfo represents indexed pool data
type PoolInfo struct {
	ID           string  `json:"id"`
	Asset0       string  `json:"asset0"`
	Asset1       string  `json:"asset1"`
	Reserve0     uint64  `json:"reserve0"`
	Reserve1     uint64  `json:"reserve1"`
	Fee          float64 `json:"fee"`
	TotalSupply  uint64  `json:"total_supply"`
}

// TokenInfo represents indexed token data
type TokenInfo struct {
	Symbol      string `json:"symbol"`
	Decimals    uint8  `json:"decimals"`
	ContractID  string `json:"contract_id"`
	Description string `json:"description"`
}

// DepositInfo represents indexed deposit data
type DepositInfo struct {
	TxID      string `json:"txid"`
	VOut      uint32 `json:"vout"`
	Amount    uint64 `json:"amount"`
	Owner     string `json:"owner"`
	Height    uint32 `json:"height"`
	Confirmed bool   `json:"confirmed"`
}

// VSCEvent represents a VSC blockchain event
type VSCEvent struct {
	Type      string          `json:"type"`
	Contract  string          `json:"contract"`
	Method    string          `json:"method"`
	Args      json.RawMessage `json:"args"`
	BlockHeight uint64        `json:"block_height"`
	TxID      string          `json:"tx_id"`
}

// NewService creates a new indexer service
func NewService(wsURL string, port string) *Service {
	svc := &Service{
		wsURL:   wsURL,
		readers: make([]ReadModel, 0),
	}

	// Add default DEX read model
	dexReader := NewDexReadModel()
	svc.AddReader(dexReader)

	// Create HTTP server
	svc.server = NewServer(svc, port)

	return svc
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

	// Start WebSocket indexing
	return s.startIndexing(ctx)
}

// startIndexing begins indexing events from VSC GraphQL subscription
func (s *Service) startIndexing(ctx context.Context) error {
	conn, _, err := websocket.DefaultDialer.Dial(s.wsURL, nil)
	if err != nil {
		return err
	}
	s.conn = conn

	defer conn.Close()

	// Subscribe to DEX and bridge events
	subscription := `{
		"type": "start",
		"payload": {
			"query": "
				subscription {
					events(filter: {contracts: [\"dex-pool\", \"btc-mapping\"]}) {
						type
						contract
						method
						args
						blockHeight
						txId
					}
				}
			"
		}
	}`

	if err := conn.WriteJSON(subscription); err != nil {
		return err
	}

	// Handle incoming events
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			var response struct {
				Type    string    `json:"type"`
				Payload VSCEvent `json:"payload"`
			}

			err := conn.ReadJSON(&response)
			if err != nil {
				log.Printf("WebSocket read error: %v", err)
				continue
			}

			if response.Type == "data" {
				s.handleEvent(response.Payload)
			}
		}
	}
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

// QueryTokens returns all indexed tokens
func (s *Service) QueryTokens() ([]TokenInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var allTokens []TokenInfo
	for _, reader := range s.readers {
		tokens, err := reader.QueryTokens()
		if err != nil {
			continue
		}
		allTokens = append(allTokens, tokens...)
	}

	return allTokens, nil
}

// QueryDeposits returns all indexed deposits
func (s *Service) QueryDeposits() ([]DepositInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var allDeposits []DepositInfo
	for _, reader := range s.readers {
		deposits, err := reader.QueryDeposits()
		if err != nil {
			continue
		}
		allDeposits = append(allDeposits, deposits...)
	}

	return allDeposits, nil
}
