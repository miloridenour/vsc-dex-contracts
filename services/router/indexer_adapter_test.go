package router

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewIndexerPoolQuerier(t *testing.T) {
	querier := NewIndexerPoolQuerier("http://localhost:8081")
	assert.NotNil(t, querier)
	assert.Equal(t, "http://localhost:8081", querier.indexerEndpoint)
	assert.NotNil(t, querier.httpClient)
}

func TestGetPoolByID_Success(t *testing.T) {
	// Create mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/pools/test-pool-123", r.URL.Path)
		assert.Equal(t, "GET", r.Method)

		// Mock indexer response (Fee as float64 percentage)
		pool := map[string]interface{}{
			"id":           "test-pool-123",
			"asset0":       "BTC",
			"asset1":       "HBD",
			"reserve0":     float64(100000000), // 1 BTC
			"reserve1":     float64(10000000),  // 10 HBD
			"fee":          0.08,                // 0.08% (will be converted to 8 basis points)
			"total_supply": float64(1000000),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(pool)
	}))
	defer server.Close()

	querier := NewIndexerPoolQuerier(server.URL)
	pool, err := querier.GetPoolByID("test-pool-123")

	require.NoError(t, err)
	assert.NotNil(t, pool)
	assert.Equal(t, "test-pool-123", pool.ID)
	assert.Equal(t, "BTC", pool.Asset0)
	assert.Equal(t, "HBD", pool.Asset1)
	assert.Equal(t, uint64(100000000), pool.Reserve0)
	assert.Equal(t, uint64(10000000), pool.Reserve1)
	assert.Equal(t, uint64(8), pool.Fee) // 0.08% = 8 basis points
}

func TestGetPoolByID_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	querier := NewIndexerPoolQuerier(server.URL)
	pool, err := querier.GetPoolByID("nonexistent-pool")

	assert.Error(t, err)
	assert.Nil(t, pool)
	assert.Contains(t, err.Error(), "pool not found")
}

func TestGetPoolByID_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	querier := NewIndexerPoolQuerier(server.URL)
	pool, err := querier.GetPoolByID("test-pool")

	assert.Error(t, err)
	assert.Nil(t, pool)
	assert.Contains(t, err.Error(), "status 500")
}

func TestGetPoolByID_MalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("invalid json{"))
	}))
	defer server.Close()

	querier := NewIndexerPoolQuerier(server.URL)
	pool, err := querier.GetPoolByID("test-pool")

	assert.Error(t, err)
	assert.Nil(t, pool)
	assert.Contains(t, err.Error(), "decode")
}

func TestGetPoolByID_NetworkError(t *testing.T) {
	// Use invalid URL to trigger network error
	querier := NewIndexerPoolQuerier("http://localhost:99999")
	pool, err := querier.GetPoolByID("test-pool")

	assert.Error(t, err)
	assert.Nil(t, pool)
	assert.Contains(t, err.Error(), "failed to query indexer")
}

func TestGetPoolsByAsset_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/pools", r.URL.Path)

		// Mock indexer response (Fee as float64 percentage)
		pools := []map[string]interface{}{
			{
				"id":       "pool-1",
				"asset0":   "BTC",
				"asset1":   "HBD",
				"reserve0": float64(100000000),
				"reserve1": float64(10000000),
				"fee":      0.08,
			},
			{
				"id":       "pool-2",
				"asset0":   "HBD",
				"asset1":   "HIVE",
				"reserve0": float64(10000000),
				"reserve1": float64(10000000),
				"fee":      0.08,
			},
			{
				"id":       "pool-3",
				"asset0":   "ETH",
				"asset1":   "HBD",
				"reserve0": float64(50000000),
				"reserve1": float64(10000000),
				"fee":      0.1,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(pools)
	}))
	defer server.Close()

	querier := NewIndexerPoolQuerier(server.URL)
	pools, err := querier.GetPoolsByAsset("BTC")

	require.NoError(t, err)
	assert.Len(t, pools, 1)
	assert.Equal(t, "pool-1", pools[0].ID)
	assert.Equal(t, "BTC", pools[0].Asset0)
}

func TestGetPoolsByAsset_FiltersCorrectly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pools := []map[string]interface{}{
			{"id": "pool-1", "asset0": "BTC", "asset1": "HBD", "reserve0": float64(100000000), "reserve1": float64(10000000), "fee": 0.08},
			{"id": "pool-2", "asset0": "HBD", "asset1": "HIVE", "reserve0": float64(10000000), "reserve1": float64(10000000), "fee": 0.08},
			{"id": "pool-3", "asset0": "HBD", "asset1": "BTC", "reserve0": float64(10000000), "reserve1": float64(100000000), "fee": 0.08}, // HBD first
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(pools)
	}))
	defer server.Close()

	querier := NewIndexerPoolQuerier(server.URL)
	pools, err := querier.GetPoolsByAsset("BTC")

	require.NoError(t, err)
	assert.Len(t, pools, 2) // pool-1 and pool-3 both contain BTC
}

func TestGetPoolsByAsset_EmptyList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]IndexerPoolInfo{})
	}))
	defer server.Close()

	querier := NewIndexerPoolQuerier(server.URL)
	pools, err := querier.GetPoolsByAsset("NONEXISTENT")

	require.NoError(t, err)
	assert.Empty(t, pools)
}

func TestGetPoolsByAsset_FeeConversion(t *testing.T) {
	testCases := []struct {
		name     string
		fee      float64
		expected uint64
	}{
		{"0.08% = 8 bps", 0.08, 8},
		{"0.1% = 10 bps", 0.1, 10},
		{"1% = 100 bps", 1.0, 100},
		{"0.5% = 50 bps", 0.5, 50},
		{"0.01% = 1 bp", 0.01, 1},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Mock indexer response (which has Fee as float64 percentage)
				pool := map[string]interface{}{
					"id":          "test-pool",
					"asset0":      "BTC",
					"asset1":      "HBD",
					"reserve0":    float64(100000000),
					"reserve1":    float64(10000000),
					"fee":         tc.fee, // float64 percentage
					"total_supply": float64(1000000),
				}

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(pool)
			}))
			defer server.Close()

			querier := NewIndexerPoolQuerier(server.URL)
			pool, err := querier.GetPoolByID("test-pool")

			require.NoError(t, err)
			assert.Equal(t, tc.expected, pool.Fee, "Fee conversion failed for %s", tc.name)
		})
	}
}

func TestGetPoolsByAsset_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	querier := NewIndexerPoolQuerier(server.URL)
	pools, err := querier.GetPoolsByAsset("BTC")

	assert.Error(t, err)
	assert.Nil(t, pools)
}

func TestGetPoolsByAsset_MalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("not json"))
	}))
	defer server.Close()

	querier := NewIndexerPoolQuerier(server.URL)
	pools, err := querier.GetPoolsByAsset("BTC")

	assert.Error(t, err)
	assert.Nil(t, pools)
}


