package indexer

import (
	"encoding/json"
	"fmt"
	"sync"
)

// DexReadModel implements read model for DEX operations
type DexReadModel struct {
	mu       sync.RWMutex
	pools    map[string]PoolInfo
	tokens   map[string]TokenInfo
	deposits map[string]DepositInfo
}

// NewDexReadModel creates a new DEX read model
func NewDexReadModel() *DexReadModel {
	return &DexReadModel{
		pools:    make(map[string]PoolInfo),
		tokens:   make(map[string]TokenInfo),
		deposits: make(map[string]DepositInfo),
	}
}

// HandleEvent processes VSC events and updates read models
func (dm *DexReadModel) HandleEvent(event VSCEvent) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	switch event.Contract {
	case "dex-pool":
		return dm.handlePoolEvent(event)
	case "token-registry":
		return dm.handleTokenEvent(event)
	case "btc-mapping":
		return dm.handleMappingEvent(event)
	}

	return nil
}

// handlePoolEvent processes pool-related events
func (dm *DexReadModel) handlePoolEvent(event VSCEvent) error {
	switch event.Method {
	case "createPool":
		var args struct {
			PoolID string  `json:"poolId"`
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

	case "addLiquidity":
		var args struct {
			PoolID  string `json:"poolId"`
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

	case "swap":
		var args struct {
			PoolID  string `json:"poolId"`
			Amount0 int64  `json:"amount0"` // Negative for output
			Amount1 int64  `json:"amount1"`
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

// handleTokenEvent processes token registry events
func (dm *DexReadModel) handleTokenEvent(event VSCEvent) error {
	switch event.Method {
	case "registerToken":
		var args struct {
			Symbol      string `json:"symbol"`
			Decimals    uint8  `json:"decimals"`
			ContractID  string `json:"contractId"`
			Description string `json:"description"`
		}
		if err := json.Unmarshal(event.Args, &args); err != nil {
			return err
		}

		dm.tokens[args.Symbol] = TokenInfo{
			Symbol:      args.Symbol,
			Decimals:    args.Decimals,
			ContractID:  args.ContractID,
			Description: args.Description,
		}
	}

	return nil
}

// handleMappingEvent processes BTC mapping events
func (dm *DexReadModel) handleMappingEvent(event VSCEvent) error {
	// Handle BTC mapping events (deposits, withdrawals, etc.)
	// These would update separate mapping-specific read models
	switch event.Method {
	case "depositMinted":
		var args struct {
			TxID      string `json:"txid"`
			VOut      uint32 `json:"vout"`
			Amount    uint64 `json:"amount"`
			Owner     string `json:"owner"`
			Height    uint32 `json:"height"`
			Confirmed bool   `json:"confirmed"`
		}
		if err := json.Unmarshal(event.Args, &args); err != nil {
			return err
		}

		key := depositKey(args.TxID, args.VOut)
		dm.deposits[key] = DepositInfo{
			TxID:      args.TxID,
			VOut:      args.VOut,
			Amount:    args.Amount,
			Owner:     args.Owner,
			Height:    args.Height,
			Confirmed: args.Confirmed,
		}
	case "depositConfirmed":
		var args struct {
			TxID   string `json:"txid"`
			VOut   uint32 `json:"vout"`
			Height uint32 `json:"height"`
		}
		if err := json.Unmarshal(event.Args, &args); err != nil {
			return err
		}

		key := depositKey(args.TxID, args.VOut)
		if deposit, exists := dm.deposits[key]; exists {
			deposit.Confirmed = true
			if args.Height > 0 {
				deposit.Height = args.Height
			}
			dm.deposits[key] = deposit
		}
	case "withdrawalRequested":
		// Withdrawal requests do not change deposit tracking directly.
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

// QueryTokens returns all indexed tokens
func (dm *DexReadModel) QueryTokens() ([]TokenInfo, error) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	tokens := make([]TokenInfo, 0, len(dm.tokens))
	for _, token := range dm.tokens {
		tokens = append(tokens, token)
	}

	return tokens, nil
}

// QueryDeposits returns deposit information
func (dm *DexReadModel) QueryDeposits() ([]DepositInfo, error) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	// Return a copy of deposits
	deposits := make([]DepositInfo, 0, len(dm.deposits))
	for _, deposit := range dm.deposits {
		deposits = append(deposits, deposit)
	}

	return deposits, nil
}

// GetDeposit returns a specific deposit by txid and vout
func (dm *DexReadModel) GetDeposit(txid string, vout uint32) (DepositInfo, bool) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	deposit, exists := dm.deposits[depositKey(txid, vout)]
	return deposit, exists
}

func depositKey(txid string, vout uint32) string {
	return fmt.Sprintf("%s:%d", txid, vout)
}

// GetPool returns a specific pool by ID
func (dm *DexReadModel) GetPool(poolID string) (PoolInfo, bool) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	pool, exists := dm.pools[poolID]
	return pool, exists
}

// GetToken returns a specific token by symbol
func (dm *DexReadModel) GetToken(symbol string) (TokenInfo, bool) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	token, exists := dm.tokens[symbol]
	return token, exists
}
