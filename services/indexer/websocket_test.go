package indexer

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

// Mock WebSocket server for testing
type mockWebSocketServer struct {
	server   *httptest.Server
	upgrader websocket.Upgrader
}

func newMockWebSocketServer() *mockWebSocketServer {
	mux := http.NewServeMux()
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	mux.HandleFunc("/graphql", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		// Handle WebSocket messages
		for {
			var msg map[string]interface{}
			err := conn.ReadJSON(&msg)
			if err != nil {
				break
			}

			// Handle connection_init
			if msg["type"] == "connection_init" {
				conn.WriteJSON(map[string]interface{}{
					"type": "connection_ack",
				})
				continue
			}

			// Handle start subscription
			if msg["type"] == "start" {
				// Send a mock event
				conn.WriteJSON(map[string]interface{}{
					"type": "data",
					"id":   msg["id"],
					"payload": map[string]interface{}{
						"data": map[string]interface{}{
							"events": []interface{}{
								map[string]interface{}{
									"type":      "contract_output",
									"contract":  "dex-router",
									"method":    "pool_created",
									"blockHeight": float64(1001),
									"txId":      "tx-123",
									"args": map[string]interface{}{
										"pool_id": "test-pool",
										"asset0":  "HBD",
										"asset1":  "HIVE",
										"fee":     0.08,
									},
								},
							},
						},
					},
				})

				// Close after sending event
				time.Sleep(10 * time.Millisecond)
				conn.WriteJSON(map[string]interface{}{
					"type": "complete",
					"id":   msg["id"],
				})
				break
			}
		}
	})

	server := httptest.NewServer(mux)

	return &mockWebSocketServer{
		server: server,
		upgrader: upgrader,
	}
}

func (m *mockWebSocketServer) URL() string {
	return "ws" + strings.TrimPrefix(m.server.URL, "http") + "/graphql"
}

func (m *mockWebSocketServer) WSURL() string {
	return "ws" + strings.TrimPrefix(m.server.URL, "http") + "/graphql"
}

func (m *mockWebSocketServer) Close() {
	m.server.Close()
}

func TestService_StartWebSocketIndexing(t *testing.T) {
	mockWS := newMockWebSocketServer()
	defer mockWS.Close()

	svc := NewService("http://localhost:4000", ":8081")
	svc.SetWebSocketURL(mockWS.URL())
	svc.SetContracts([]string{"dex-router"})

	// Add a reader to capture events
	eventCaptured := make(chan VSCEvent, 1)
	mockReader := &mockReadModel{eventChan: eventCaptured}
	svc.AddReader(mockReader)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := svc.startWebSocketIndexing(ctx)

	// Should complete without error (WebSocket connection closes normally)
	assert.NoError(t, err)

	// Verify event was processed
	select {
	case event := <-eventCaptured:
		assert.Equal(t, "contract_output", event.Type)
		assert.Equal(t, "dex-router", event.Contract)
		assert.Equal(t, uint64(1001), event.BlockHeight)
		assert.Equal(t, "tx-123", event.TxID)
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Event was not processed")
	}
}

func TestService_StartWebSocketIndexing_ConnectionFailure(t *testing.T) {
	svc := NewService("http://localhost:4000", ":8081")

	// Try to connect to invalid WebSocket URL
	svc.SetWebSocketURL("ws://invalid-host:9999/graphql")

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := svc.startWebSocketIndexing(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "WebSocket connection failed")
}

func TestService_StartWebSocketIndexing_InvalidAck(t *testing.T) {
	mux := http.NewServeMux()
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	mux.HandleFunc("/graphql", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		// Read connection_init
		var msg map[string]interface{}
		conn.ReadJSON(&msg)

		// Send invalid ack
		conn.WriteJSON(map[string]interface{}{
			"type": "invalid_ack",
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/graphql"

	svc := NewService("http://localhost:4000", ":8081")
	svc.SetWebSocketURL(wsURL)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	err := svc.startWebSocketIndexing(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected connection_ack")
}

func TestService_Start(t *testing.T) {
	// Test that Start method works with polling (WebSocket disabled)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/graphql" {
			var req map[string]interface{}
			json.NewDecoder(r.Body).Decode(&req)

			if strings.Contains(r.URL.RawQuery, "last_processed_block") {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"data": map[string]interface{}{
						"localNodeInfo": map[string]interface{}{
							"last_processed_block": float64(1000),
						},
					},
				})
			}
		}
	}))
	defer server.Close()

	svc := NewService(server.URL, ":8081")
	svc.SetContracts([]string{"dex-router"})

	// Start should begin polling since WebSocket is disabled
	// We can't actually start the service since it would run indefinitely,
	// but we can test that the components are initialized properly

	// Verify service is properly configured
	assert.NotNil(t, svc.httpURL)
	assert.NotNil(t, svc.readers)
	assert.Len(t, svc.readers, 1)
	assert.False(t, svc.useWebSocket)
}

func TestService_ErrorHandling_PollForEvents(t *testing.T) {
	svc := NewService("http://invalid-host:9999", ":8081")

	// This should fail gracefully
	err := svc.pollForEvents(context.Background())
	assert.Error(t, err)
}

func TestService_ErrorHandling_UpdateLastBlock(t *testing.T) {
	svc := NewService("http://invalid-host:9999", ":8081")

	err := svc.updateLastBlock(context.Background())
	assert.Error(t, err)
}
