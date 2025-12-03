package indexer

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewService(t *testing.T) {
	svc := NewService("http://localhost:4000", ":8081")
	assert.NotNil(t, svc)
	assert.Equal(t, "http://localhost:4000", svc.httpURL)
	assert.NotNil(t, svc.readers)
	assert.Len(t, svc.readers, 1) // Should have default DEX read model

	// Verify default read model is DexReadModel
	dexReader, ok := svc.readers[0].(*DexReadModel)
	assert.True(t, ok)
	assert.NotNil(t, dexReader)
}

func TestService_SetContracts(t *testing.T) {
	svc := NewService("http://localhost:4000", ":8081")

	contracts := []string{"dex-router", "other-contract"}
	svc.SetContracts(contracts)

	// Verify contracts are set (need to access private field via method if available)
	// For now, test that the method doesn't panic
	assert.NotPanics(t, func() {
		svc.SetContracts(contracts)
	})
}

func TestService_AddReader(t *testing.T) {
	svc := NewService("http://localhost:4000", ":8081")

	customReader := &DexReadModel{} // Could be a mock reader
	svc.AddReader(customReader)

	assert.Len(t, svc.readers, 2) // Default + custom
	assert.Contains(t, svc.readers, customReader)
}

func TestService_QueryPools(t *testing.T) {
	svc := NewService("http://localhost:4000", ":8081")

	// Add a pool to the default DEX reader
	dexReader := svc.readers[0].(*DexReadModel)
	dexReader.pools["test-pool"] = PoolInfo{
		ID:       "test-pool",
		Asset0:   "HBD",
		Asset1:   "HIVE",
		Reserve0: 1000000,
		Reserve1: 500000,
	}

	pools, err := svc.QueryPools()
	require.NoError(t, err)
	assert.Len(t, pools, 1)
	assert.Equal(t, "test-pool", pools[0].ID)
}

// Mock HTTP server for testing GraphQL queries
type mockGraphQLServer struct {
	server *httptest.Server
}

func newMockGraphQLServer() *mockGraphQLServer {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/graphql", func(w http.ResponseWriter, r *http.Request) {
		var req map[string]interface{}
		json.NewDecoder(r.Body).Decode(&req)

		query := req["query"].(string)

		var response interface{}

		if bytes.Contains([]byte(query), []byte("last_processed_block")) {
			response = map[string]interface{}{
				"data": map[string]interface{}{
					"localNodeInfo": map[string]interface{}{
						"last_processed_block": float64(1000),
					},
				},
			}
		} else if bytes.Contains([]byte(query), []byte("findContractOutput")) {
			response = map[string]interface{}{
				"data": map[string]interface{}{
					"findContractOutput": []interface{}{
						map[string]interface{}{
							"id":          "tx-123",
							"block_height": float64(1001),
							"contract_id":  "dex-router",
							"results": []interface{}{
								map[string]interface{}{
									"ret": `{"type": "pool_created", "pool_id": "pool-123"}`,
									"ok":  true,
								},
							},
						},
					},
				},
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	server := httptest.NewServer(mux)
	return &mockGraphQLServer{server: server}
}

func (m *mockGraphQLServer) URL() string {
	return m.server.URL
}

func (m *mockGraphQLServer) Close() {
	m.server.Close()
}

func TestService_StartPolling(t *testing.T) {
	mockServer := newMockGraphQLServer()
	defer mockServer.Close()

	svc := NewService(mockServer.URL(), ":8081")
	svc.SetContracts([]string{"dex-router"})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Start polling in background
	errCh := make(chan error, 1)
	go func() {
		errCh <- svc.startPolling(ctx)
	}()

	// Wait a bit for polling to happen
	time.Sleep(500 * time.Millisecond)

	// Cancel context to stop polling
	cancel()

	// Should complete without error
	select {
	case err := <-errCh:
		assert.NoError(t, err)
	case <-time.After(1 * time.Second):
		t.Fatal("startPolling didn't complete")
	}
}

func TestService_UpdateLastBlock(t *testing.T) {
	mockServer := newMockGraphQLServer()
	defer mockServer.Close()

	svc := NewService(mockServer.URL(), ":8081")

	err := svc.updateLastBlock(context.Background())
	require.NoError(t, err)

	// Verify last block was updated
	assert.Equal(t, uint64(1000), svc.lastBlock)
}

func TestService_PollContractOutputs(t *testing.T) {
	mockServer := newMockGraphQLServer()
	defer mockServer.Close()

	svc := NewService(mockServer.URL(), ":8081")
	svc.SetContracts([]string{"dex-router"})

	// Add a reader to capture events
	eventCaptured := make(chan VSCEvent, 1)
	mockReader := &mockReadModel{eventChan: eventCaptured}
	svc.AddReader(mockReader)

	err := svc.pollContractOutputs(context.Background(), "dex-router", 999)
	require.NoError(t, err)

	// Wait for event to be processed
	select {
	case event := <-eventCaptured:
		assert.Equal(t, "contract_output", event.Type)
		assert.Equal(t, "dex-router", event.Contract)
		assert.Equal(t, uint64(1001), event.BlockHeight)
	case <-time.After(1 * time.Second):
		t.Fatal("Event was not processed")
	}
}

// Mock reader for testing
type mockReadModel struct {
	eventChan chan VSCEvent
}

func (m *mockReadModel) HandleEvent(event VSCEvent) error {
	select {
	case m.eventChan <- event:
	default:
	}
	return nil
}

func (m *mockReadModel) QueryPools() ([]PoolInfo, error) {
	return []PoolInfo{}, nil
}

// Test WebSocket connection setup (mock)
func TestService_SetWebSocketURL(t *testing.T) {
	svc := NewService("http://localhost:4000", ":8081")

	// Initially no WebSocket
	assert.False(t, svc.useWebSocket)
	assert.Empty(t, svc.wsURL)

	// Set WebSocket URL
	svc.SetWebSocketURL("ws://localhost:4000/graphql")

	assert.True(t, svc.useWebSocket)
	assert.Equal(t, "ws://localhost:4000/graphql", svc.wsURL)
}

// Test parseVSCEvent function
func TestParseVSCEvent(t *testing.T) {
	eventMap := map[string]interface{}{
		"type":        "swap",
		"contract":    "dex-router",
		"method":      "execute",
		"txId":        "tx-123",
		"blockHeight": float64(1001),
		"args": map[string]interface{}{
			"pool_id": "pool-123",
			"amount0": float64(1000),
		},
	}

	event := parseVSCEvent(eventMap)
	require.NotNil(t, event)
	assert.Equal(t, "swap", event.Type)
	assert.Equal(t, "dex-router", event.Contract)
	assert.Equal(t, "execute", event.Method)
	assert.Equal(t, "tx-123", event.TxID)
	assert.Equal(t, uint64(1001), event.BlockHeight)

	// Verify args were marshaled to JSON
	expectedArgs := `{"amount0":1000,"pool_id":"pool-123"}`
	assert.JSONEq(t, expectedArgs, string(event.Args))
}
