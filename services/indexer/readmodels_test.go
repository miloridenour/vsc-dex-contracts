package indexer

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDexReadModel(t *testing.T) {
	rm := NewDexReadModel()
	assert.NotNil(t, rm)
	assert.NotNil(t, rm.pools)
	assert.Empty(t, rm.pools)
}

func TestDexReadModel_HandleEvent_PoolCreated(t *testing.T) {
	rm := NewDexReadModel()

	event := VSCEvent{
		Type:     "contract_output",
		Contract: "dex-router",
		Method:   "pool_created",
		Args: json.RawMessage(`{
			"pool_id": "pool-123",
			"asset0": "HBD",
			"asset1": "HIVE",
			"fee": 0.08
		}`),
	}

	err := rm.HandleEvent(event)
	require.NoError(t, err)

	// Verify pool was created
	pool, exists := rm.pools["pool-123"]
	assert.True(t, exists)
	assert.Equal(t, "pool-123", pool.ID)
	assert.Equal(t, "HBD", pool.Asset0)
	assert.Equal(t, "HIVE", pool.Asset1)
	assert.Equal(t, 0.08, pool.Fee)
	assert.Equal(t, uint64(0), pool.Reserve0)
	assert.Equal(t, uint64(0), pool.Reserve1)
}

func TestDexReadModel_HandleEvent_LiquidityAdded(t *testing.T) {
	rm := NewDexReadModel()

	// First create a pool
	event := VSCEvent{
		Type:     "contract_output",
		Contract: "dex-router",
		Method:   "pool_created",
		Args: json.RawMessage(`{
			"pool_id": "pool-123",
			"asset0": "HBD",
			"asset1": "HIVE",
			"fee": 0.08
		}`),
	}
	rm.HandleEvent(event)

	// Then add liquidity
	liquidityEvent := VSCEvent{
		Type:     "contract_output",
		Contract: "dex-router",
		Method:   "liquidity_added",
		Args: json.RawMessage(`{
			"pool_id": "pool-123",
			"amount0": 1000000,
			"amount1": 500000
		}`),
	}

	err := rm.HandleEvent(liquidityEvent)
	require.NoError(t, err)

	// Verify reserves were updated
	pool, exists := rm.pools["pool-123"]
	assert.True(t, exists)
	assert.Equal(t, uint64(1000000), pool.Reserve0)
	assert.Equal(t, uint64(500000), pool.Reserve1)
	assert.Equal(t, uint64(1000000), pool.TotalSupply) // Simplified LP calculation
}

func TestDexReadModel_HandleEvent_SwapExecuted(t *testing.T) {
	rm := NewDexReadModel()

	// Create pool with initial reserves
	rm.pools["pool-123"] = PoolInfo{
		ID:          "pool-123",
		Asset0:      "HBD",
		Asset1:      "HIVE",
		Reserve0:    1000000,
		Reserve1:    500000,
		TotalSupply: 1000000,
	}

	// Execute swap (reserve changes from AMM calculation)
	swapEvent := VSCEvent{
		Type:     "contract_output",
		Contract: "dex-router",
		Method:   "swap_executed",
		Args: json.RawMessage(`{
			"pool_id": "pool-123",
			"amount0": -100000,
			"amount1": 25000
		}`),
	}

	err := rm.HandleEvent(swapEvent)
	require.NoError(t, err)

	// Verify reserves were updated correctly
	pool, exists := rm.pools["pool-123"]
	assert.True(t, exists)
	assert.Equal(t, uint64(900000), pool.Reserve0)  // 1000000 - 100000
	assert.Equal(t, uint64(525000), pool.Reserve1)  // 500000 + 25000
}

func TestDexReadModel_HandleEvent_InvalidJSON(t *testing.T) {
	rm := NewDexReadModel()

	event := VSCEvent{
		Type:     "contract_output",
		Contract: "dex-router",
		Method:   "pool_created",
		Args:     json.RawMessage(`invalid json{`),
	}

	err := rm.HandleEvent(event)
	assert.Error(t, err)
}

func TestDexReadModel_HandleEvent_NonDexContract(t *testing.T) {
	rm := NewDexReadModel()

	event := VSCEvent{
		Type:     "contract_output",
		Contract: "other-contract",
		Method:   "some_method",
		Args:     json.RawMessage(`{}`),
	}

	err := rm.HandleEvent(event)
	assert.NoError(t, err) // Should ignore non-dex contracts
	assert.Empty(t, rm.pools)
}

func TestDexReadModel_QueryPools(t *testing.T) {
	rm := NewDexReadModel()

	// Add some test pools
	rm.pools["pool-1"] = PoolInfo{ID: "pool-1", Asset0: "HBD", Asset1: "HIVE"}
	rm.pools["pool-2"] = PoolInfo{ID: "pool-2", Asset0: "BTC", Asset1: "HBD"}

	pools, err := rm.QueryPools()
	require.NoError(t, err)
	assert.Len(t, pools, 2)

	// Verify pools contain expected data
	poolIDs := make([]string, len(pools))
	for i, pool := range pools {
		poolIDs[i] = pool.ID
	}
	assert.Contains(t, poolIDs, "pool-1")
	assert.Contains(t, poolIDs, "pool-2")
}

func TestDexReadModel_GetPool(t *testing.T) {
	rm := NewDexReadModel()

	testPool := PoolInfo{
		ID:       "test-pool",
		Asset0:   "HBD",
		Asset1:   "HIVE",
		Reserve0: 1000000,
		Reserve1: 500000,
		Fee:      0.08,
	}

	rm.pools["test-pool"] = testPool

	// Test existing pool
	pool, exists := rm.GetPool("test-pool")
	assert.True(t, exists)
	assert.Equal(t, testPool, pool)

	// Test non-existing pool
	pool, exists = rm.GetPool("nonexistent")
	assert.False(t, exists)
	assert.Equal(t, PoolInfo{}, pool)
}
