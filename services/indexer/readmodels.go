package indexer

import (
	"encoding/json"
	"sync"
)

// DexReadModel implements read model for DEX operations
type DexReadModel struct {
	mu    sync.RWMutex
	pools map[string]PoolInfo
}

// NewDexReadModel creates a new DEX read model
func NewDexReadModel() *DexReadModel {
	return &DexReadModel{
		pools: make(map[string]PoolInfo),
	}
}

// HandleEvent processes VSC events and updates read models
func (dm *DexReadModel) HandleEvent(event VSCEvent) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	switch event.Contract {
	case "dex-router":
		return dm.handleDexRouterEvent(event)
	}

	return nil
}

// handleDexRouterEvent processes DEX router events
func (dm *DexReadModel) handleDexRouterEvent(event VSCEvent) error {
	// Handle pool creation, liquidity changes, and swaps from unified contract
	switch event.Method {
	case "pool_created":
		var args struct {
			PoolID string  `json:"pool_id"`
			Asset0 string  `json:"asset0"`
			Asset1 string  `json:"asset1"`
			Fee    float64 `json:"fee"`
		}
		if err := json.Unmarshal(event.Args, &args); err != nil {
			return err
		}

		dm.pools[args.PoolID] = PoolInfo{
			ID:       args.PoolID,
			Asset0:   args.Asset0,
			Asset1:   args.Asset1,
			Fee:      args.Fee,
			Reserve0: 0,
			Reserve1: 0,
		}
	case "liquidity_added":
		var args struct {
			PoolID  string `json:"pool_id"`
			Amount0 uint64 `json:"amount0"`
			Amount1 uint64 `json:"amount1"`
		}
		if err := json.Unmarshal(event.Args, &args); err != nil {
			return err
		}

		if pool, exists := dm.pools[args.PoolID]; exists {
			pool.Reserve0 += args.Amount0
			pool.Reserve1 += args.Amount1
			pool.TotalSupply += args.Amount0 // Simplified LP token calculation
			dm.pools[args.PoolID] = pool
		}
	case "swap_executed":
		var args struct {
			PoolID  string `json:"pool_id"`
			Amount0 int64  `json:"amount0"` // Reserve change for asset0
			Amount1 int64  `json:"amount1"` // Reserve change for asset1
		}
		if err := json.Unmarshal(event.Args, &args); err != nil {
			return err
		}

		if pool, exists := dm.pools[args.PoolID]; exists {
			pool.Reserve0 = uint64(int64(pool.Reserve0) + args.Amount0)
			pool.Reserve1 = uint64(int64(pool.Reserve1) + args.Amount1)
			dm.pools[args.PoolID] = pool
		}
	}

	return nil
}



// QueryPools returns all indexed pools
func (dm *DexReadModel) QueryPools() ([]PoolInfo, error) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	pools := make([]PoolInfo, 0, len(dm.pools))
	for _, pool := range dm.pools {
		pools = append(pools, pool)
	}

	return pools, nil
}

// GetPool returns a specific pool by ID
func (dm *DexReadModel) GetPool(poolID string) (PoolInfo, bool) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	pool, exists := dm.pools[poolID]
	return pool, exists
}

