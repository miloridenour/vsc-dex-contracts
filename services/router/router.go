package router

import (
	"context"
	"fmt"
	"log"
	"math/big"
)

// DEXExecutor interface for executing DEX swaps
type DEXExecutor interface {
	ExecuteDexSwap(ctx context.Context, amountOut int64, route []string, fee int64) error
}

// RouteResult represents the result of a route computation for DEX execution
type RouteResult struct {
	AmountOut   int64    `json:"amount_out"`
	Route       []string `json:"route"`
	PriceImpact float64  `json:"price_impact"`
	Fee         int64    `json:"fee"`
}

// PoolQuerier interface for querying pool information
type PoolQuerier interface {
	GetPoolByID(poolID string) (*PoolInfoWithReserves, error)
	GetPoolsByAsset(asset string) ([]PoolInfoWithReserves, error)
}

// Service provides DEX routing and transaction composition
type Service struct {
	vscConfig   VSCConfig
	adapters    *ChainAdapters
	poolQuerier PoolQuerier
	dexExecutor DEXExecutor
}

type VSCConfig struct {
	Endpoint string
	Key      string
	Username string
}

// SwapParams represents a swap request
type SwapParams struct {
	Sender         string
	AmountIn       int64
	AssetIn        string
	AssetOut       string
	MinAmountOut   int64
	MaxSlippage    uint64
	MiddleOutRatio float64
	Beneficiary    string
	RefBps         uint64
}

// SwapResult represents the result of a route computation
type SwapResult struct {
	Success      bool
	AmountOut    int64
	HbdAmount    int64
	Fee0         int64
	Fee1         int64
	Route        []string
	ErrorMessage string
}

// PoolHop represents a hop in a swap route
type PoolHop struct {
	PoolID    string
	TokenIn   string
	TokenOut  string
	AmountIn  int64
	AmountOut int64
}

// PoolInfo represents pool information
type PoolInfo struct {
	ContractId string
	Asset0     string
	Asset1     string
}

// PoolInfoWithReserves represents pool information with reserve data
type PoolInfoWithReserves struct {
	ContractId string
	Asset0     string
	Asset1     string
	Reserve0   uint64
	Reserve1   uint64
	Fee        uint64 // Fee in basis points (e.g., 8 = 0.08%)
}

// GeneratePairAccount generates a DEX pair account for a specific chain
func (r *Service) GeneratePairAccount(asset0, asset1, chain string) string {
	// Ensure assets are in consistent order (alphabetical)
	if asset0 > asset1 {
		asset0, asset1 = asset1, asset0
	}
	return fmt.Sprintf("%s:dex-pair-%s-%s", chain, asset0, asset1)
}

// ExecuteSwap orchestrates a two-hop swap through HBD
func (r *Service) ExecuteSwap(params SwapParams) (*SwapResult, error) {
	// Validate that we're not trying to swap the same asset
	if params.AssetIn == params.AssetOut {
		return &SwapResult{
			Success:      false,
			ErrorMessage: "cannot swap asset to itself",
		}, nil
	}

	// If already swapping to/from HBD, do direct swap
	if params.AssetIn == "HBD" || params.AssetOut == "HBD" {
		return r.executeDirectSwap(params)
	}

	// Find pools for two-hop swap
	pool1, err := r.findPool(params.AssetIn, "HBD")
	if err != nil {
		return &SwapResult{
			Success:      false,
			ErrorMessage: fmt.Sprintf("no pool found for %s/HBD: %v", params.AssetIn, err),
		}, nil
	}

	pool2, err := r.findPool("HBD", params.AssetOut)
	if err != nil {
		return &SwapResult{
			Success:      false,
			ErrorMessage: fmt.Sprintf("no pool found for HBD/%s: %v", params.AssetOut, err),
		}, nil
	}

	// Calculate expected outputs and slippage bounds
	expectedHbdOut, err := r.calculateExpectedOutput(params.AmountIn, pool1.ContractId, params.AssetIn, "HBD")
	if err != nil {
		return &SwapResult{
			Success:      false,
			ErrorMessage: fmt.Sprintf("failed to calculate HBD output: %v", err),
		}, nil
	}

	expectedFinalOut, err := r.calculateExpectedOutput(expectedHbdOut, pool2.ContractId, "HBD", params.AssetOut)
	if err != nil {
		return &SwapResult{
			Success:      false,
			ErrorMessage: fmt.Sprintf("failed to calculate final output: %v", err),
		}, nil
	}

	// Calculate slippage bounds
	minHbdOut := r.calculateMinOutput(expectedHbdOut, params.MaxSlippage, params.MiddleOutRatio)
	minFinalOut := r.calculateMinOutput(expectedFinalOut, params.MaxSlippage, 1.0)

	// Use the higher of user-specified minimum or calculated minimum
	if params.MinAmountOut > minFinalOut {
		minFinalOut = params.MinAmountOut
	}

	// Execute first swap: AssetIn -> HBD
	firstSwapParams := SwapParams{
		Sender:       params.Sender,
		AmountIn:     params.AmountIn,
		AssetIn:      params.AssetIn,
		AssetOut:     "HBD",
		MinAmountOut: minHbdOut,
		MaxSlippage:  params.MaxSlippage,
		Beneficiary:  params.Beneficiary,
		RefBps:       params.RefBps,
	}

	firstResult, err := r.executeDirectSwap(firstSwapParams)
	if err != nil {
		return &SwapResult{
			Success:      false,
			ErrorMessage: fmt.Sprintf("first swap error: %v", err),
			Route:        []string{},
		}, nil
	}
	if !firstResult.Success {
		return &SwapResult{
			Success:      false,
			ErrorMessage: fmt.Sprintf("first swap failed: %s", firstResult.ErrorMessage),
			Route:        []string{},
		}, nil
	}

	// Execute second swap: HBD -> AssetOut
	secondSwapParams := SwapParams{
		Sender:       params.Sender,
		AmountIn:     firstResult.HbdAmount,
		AssetIn:      "HBD",
		AssetOut:     params.AssetOut,
		MinAmountOut: minFinalOut,
		MaxSlippage:  params.MaxSlippage,
		Beneficiary:  params.Beneficiary,
		RefBps:       params.RefBps,
	}

	secondResult, err := r.executeDirectSwap(secondSwapParams)
	if err != nil {
		return &SwapResult{
			Success:      false,
			ErrorMessage: fmt.Sprintf("second swap failed: %v", err),
		}, nil
	}
	if !secondResult.Success {
		return &SwapResult{
			Success:      false,
			ErrorMessage: fmt.Sprintf("second swap failed: %s", secondResult.ErrorMessage),
		}, nil
	}

	return &SwapResult{
		AmountOut: secondResult.AmountOut,
		HbdAmount: firstResult.HbdAmount,
		Fee0:      firstResult.Fee0,
		Fee1:      secondResult.Fee0,
		Route:     []string{"hive", "hbd-intermediate"}, // Two-hop route through HBD
		Success:   true,
	}, nil
}

// executeDirectSwap executes a single swap between two assets
func (r *Service) executeDirectSwap(params SwapParams) (*SwapResult, error) {
	// Validate input amount
	if params.AmountIn <= 0 {
		return &SwapResult{
			Success:      false,
			ErrorMessage: "amount in must be greater than zero",
		}, nil
	}

	// Validate amount doesn't exceed maximum safe value
	maxSafeAmount := int64(1 << 62) // Half of max int64 to prevent overflow
	if params.AmountIn > maxSafeAmount {
		return &SwapResult{
			Success:      false,
			ErrorMessage: "swap amount too large",
		}, nil
	}

	pool, err := r.findPool(params.AssetIn, params.AssetOut)
	if err != nil {
		return &SwapResult{
			Success:      false,
			ErrorMessage: fmt.Sprintf("no pool found for %s/%s: %v", params.AssetIn, params.AssetOut, err),
		}, nil
	}

	// Calculate actual swap output using AMM formula
	poolWithReserves, err := r.getPoolWithReserves(pool.ContractId)
	if err != nil {
		return &SwapResult{
			Success:      false,
			ErrorMessage: fmt.Sprintf("failed to get pool reserves: %v", err),
		}, nil
	}

	// Determine which asset is which in the pool
	var reserveIn, reserveOut uint64
	if poolWithReserves.Asset0 == params.AssetIn {
		reserveIn = poolWithReserves.Reserve0
		reserveOut = poolWithReserves.Reserve1
	} else {
		reserveIn = poolWithReserves.Reserve1
		reserveOut = poolWithReserves.Reserve0
	}

	// Validate reserves
	if reserveIn == 0 || reserveOut == 0 {
		return &SwapResult{
			Success:      false,
			ErrorMessage: "pool has zero reserves",
		}, nil
	}

	// Pool drain protection: prevent swapping more than 50% of reserve
	maxSwapPercent := uint64(50) // 50% maximum
	maxAllowedAmount := reserveIn * maxSwapPercent / 100
	amountInUint64 := uint64(params.AmountIn)
	if amountInUint64 > maxAllowedAmount {
		return &SwapResult{
			Success:      false,
			ErrorMessage: fmt.Sprintf("swap amount exceeds maximum allowed (50%% of reserve): max %d, requested %d", maxAllowedAmount, amountInUint64),
		}, nil
	}

	// Calculate output using constant product formula: (x + dx) * (y - dy) = x * y
	// Solving for dy: dy = y - (x * y) / (x + dx)
	// Apply fee (fee is in basis points, e.g., 8 = 0.08% = 0.0008)
	feeBps := poolWithReserves.Fee
	if feeBps == 0 {
		feeBps = 8 // Default 0.08% fee
	}
	if feeBps >= 10000 {
		return &SwapResult{
			Success:      false,
			ErrorMessage: "invalid fee: must be less than 100%",
		}, nil
	}

	feeMultiplier := uint64(10000 - feeBps)
	amountInAfterFee := amountInUint64 * feeMultiplier / 10000
	if amountInAfterFee == 0 {
		return &SwapResult{
			Success:      false,
			ErrorMessage: "swap amount too small after fees",
		}, nil
	}

	// Use big.Int to prevent overflow in constant product calculation
	// Constant product: k = reserveIn * reserveOut
	// After swap: (reserveIn + amountInAfterFee) * (reserveOut - amountOut) = k
	// amountOut = reserveOut - k / (reserveIn + amountInAfterFee)
	reserveInBig := new(big.Int).SetUint64(reserveIn)
	reserveOutBig := new(big.Int).SetUint64(reserveOut)
	amountInAfterFeeBig := new(big.Int).SetUint64(amountInAfterFee)

	// k = reserveIn * reserveOut
	k := new(big.Int).Mul(reserveInBig, reserveOutBig)

	// newReserveIn = reserveIn + amountInAfterFee
	newReserveInBig := new(big.Int).Add(reserveInBig, amountInAfterFeeBig)

	// amountOut = reserveOut - k / newReserveIn
	// k / newReserveIn
	kDivNewReserveIn := new(big.Int).Div(k, newReserveInBig)

	// amountOut = reserveOut - (k / newReserveIn)
	amountOutBig := new(big.Int).Sub(reserveOutBig, kDivNewReserveIn)

	// Validate amountOut is positive and within bounds
	if amountOutBig.Sign() <= 0 || !amountOutBig.IsUint64() {
		return &SwapResult{
			Success:      false,
			ErrorMessage: "invalid swap output calculation: result out of bounds",
		}, nil
	}

	amountOut := amountOutBig.Uint64()

	// Validate output is positive and less than reserve
	if amountOut == 0 {
		return &SwapResult{
			Success:      false,
			ErrorMessage: "swap output is zero",
		}, nil
	}
	if amountOut >= reserveOut {
		return &SwapResult{
			Success:      false,
			ErrorMessage: "invalid swap output calculation: output exceeds reserve",
		}, nil
	}

	// Check minimum output requirement
	if params.MinAmountOut > 0 && int64(amountOut) < params.MinAmountOut {
		return &SwapResult{
			Success:      false,
			ErrorMessage: fmt.Sprintf("slippage tolerance exceeded: expected at least %d, got %d", params.MinAmountOut, amountOut),
		}, nil
	}

	// Calculate fee
	fee := params.AmountIn - int64(amountInAfterFee)

	hbdAmount := int64(amountOut)
	if params.AssetOut == "HBD" {
		hbdAmount = int64(amountOut)
	} else if params.AssetIn == "HBD" {
		hbdAmount = params.AmountIn
	}

	return &SwapResult{
		AmountOut: int64(amountOut),
		HbdAmount: hbdAmount,
		Fee0:      fee,
		Route:     []string{"hive"}, // Direct swap
		Success:   true,
	}, nil
}

// findPool finds a pool for the given asset pair
func (r *Service) findPool(asset0, asset1 string) (*PoolInfo, error) {
	// If we have a pool querier, use it to find pools
	if r.poolQuerier != nil {
		pools, err := r.poolQuerier.GetPoolsByAsset(asset0)
		if err == nil {
			for _, pool := range pools {
				if (pool.Asset0 == asset0 && pool.Asset1 == asset1) ||
					(pool.Asset0 == asset1 && pool.Asset1 == asset0) {
					return &PoolInfo{
						ContractId: pool.ContractId,
						Asset0:     pool.Asset0,
						Asset1:     pool.Asset1,
					}, nil
				}
			}
		}
	}

	// Fallback to hardcoded pools for testing
	if (asset0 == "BTC" && asset1 == "HBD") || (asset0 == "HBD" && asset1 == "BTC") {
		return &PoolInfo{
			ContractId: "btc-hbd-pool",
			Asset0:     "BTC",
			Asset1:     "HBD",
		}, nil
	}

	if (asset0 == "HBD_SAVINGS" && asset1 == "HBD") || (asset0 == "HBD" && asset1 == "HBD_SAVINGS") {
		return &PoolInfo{
			ContractId: "hbd-savings-hbd-pool",
			Asset0:     "HBD_SAVINGS",
			Asset1:     "HBD",
		}, nil
	}

	if (asset0 == "HBD" && asset1 == "HIVE") || (asset0 == "HIVE" && asset1 == "HBD") {
		return &PoolInfo{
			ContractId: "hive-hbd-pool",
			Asset0:     "HBD",
			Asset1:     "HIVE",
		}, nil
	}

	return nil, fmt.Errorf("no pool found for %s/%s", asset0, asset1)
}

// calculateExpectedOutput calculates expected output for a swap using AMM formula
func (r *Service) calculateExpectedOutput(amountIn int64, contractId, assetIn, assetOut string) (int64, error) {
	if amountIn <= 0 {
		return 0, fmt.Errorf("amount in must be greater than zero")
	}

	pool, err := r.getPoolWithReserves(contractId)
	if err != nil {
		return 0, fmt.Errorf("failed to get pool reserves: %w", err)
	}

	// Determine which asset is which in the pool
	var reserveIn, reserveOut uint64
	if pool.Asset0 == assetIn {
		reserveIn = pool.Reserve0
		reserveOut = pool.Reserve1
	} else if pool.Asset1 == assetIn {
		reserveIn = pool.Reserve1
		reserveOut = pool.Reserve0
	} else {
		return 0, fmt.Errorf("asset %s not found in pool", assetIn)
	}

	// Validate reserves
	if reserveIn == 0 || reserveOut == 0 {
		return 0, fmt.Errorf("pool has zero reserves")
	}

	// Apply fee (fee is in basis points)
	feeBps := pool.Fee
	if feeBps == 0 {
		feeBps = 8 // Default 0.08% fee
	}
	feeMultiplier := uint64(10000 - feeBps)
	amountInAfterFee := uint64(amountIn) * feeMultiplier / 10000
	if amountInAfterFee == 0 {
		amountInAfterFee = 1
	}

	// Use big.Int to prevent overflow in constant product calculation
	// Constant product: k = reserveIn * reserveOut
	// After swap: (reserveIn + amountInAfterFee) * (reserveOut - amountOut) = k
	// amountOut = reserveOut - k / (reserveIn + amountInAfterFee)
	reserveInBig := new(big.Int).SetUint64(reserveIn)
	reserveOutBig := new(big.Int).SetUint64(reserveOut)
	amountInAfterFeeBig := new(big.Int).SetUint64(amountInAfterFee)

	// k = reserveIn * reserveOut
	k := new(big.Int).Mul(reserveInBig, reserveOutBig)

	// newReserveIn = reserveIn + amountInAfterFee
	newReserveInBig := new(big.Int).Add(reserveInBig, amountInAfterFeeBig)

	// amountOut = reserveOut - k / newReserveIn
	kDivNewReserveIn := new(big.Int).Div(k, newReserveInBig)
	amountOutBig := new(big.Int).Sub(reserveOutBig, kDivNewReserveIn)

	if amountOutBig.Sign() <= 0 || !amountOutBig.IsUint64() {
		return 0, fmt.Errorf("invalid swap output calculation: result out of bounds")
	}

	amountOut := amountOutBig.Uint64()
	if amountOut >= reserveOut {
		return 0, fmt.Errorf("invalid swap output calculation: output exceeds reserve")
	}

	return int64(amountOut), nil
}

// getPoolWithReserves retrieves pool information with reserves
func (r *Service) getPoolWithReserves(contractId string) (*PoolInfoWithReserves, error) {
	// If we have a pool querier, use it
	if r.poolQuerier != nil {
		return r.poolQuerier.GetPoolByID(contractId)
	}

	// Fallback to hardcoded pools with default reserves for testing
	// In production, this should always use the pool querier
	return r.getDefaultPoolWithReserves(contractId)
}

// getDefaultPoolWithReserves returns default pool reserves for testing
func (r *Service) getDefaultPoolWithReserves(contractId string) (*PoolInfoWithReserves, error) {
	switch contractId {
	case "btc-hbd-pool":
		return &PoolInfoWithReserves{
			ContractId: "btc-hbd-pool",
			Asset0:     "BTC",
			Asset1:     "HBD",
			Reserve0:   100000000, // 1 BTC
			Reserve1:   10000000,  // 10 HBD
			Fee:        8,         // 0.08%
		}, nil
	case "hbd-savings-hbd-pool":
		return &PoolInfoWithReserves{
			ContractId: "hbd-savings-hbd-pool",
			Asset0:     "HBD_SAVINGS",
			Asset1:     "HBD",
			Reserve0:   10000000, // 10 HBD_SAVINGS
			Reserve1:   10000000, // 10 HBD
			Fee:        8,
		}, nil
	case "hive-hbd-pool":
		return &PoolInfoWithReserves{
			ContractId: "hive-hbd-pool",
			Asset0:     "HBD",
			Asset1:     "HIVE",
			Reserve0:   10000000, // 10 HBD
			Reserve1:   10000000, // 10 HIVE
			Fee:        8,
		}, nil
	default:
		return nil, fmt.Errorf("pool not found: %s", contractId)
	}
}

// calculateMinOutput calculates minimum output based on slippage tolerance
// Uses integer arithmetic to avoid floating point precision loss
func (r *Service) calculateMinOutput(expectedOut int64, maxSlippageBps uint64, ratio float64) int64 {
	// Validate inputs
	if expectedOut <= 0 {
		return 0
	}
	if maxSlippageBps > 10000 {
		maxSlippageBps = 10000 // Cap at 100%
	}
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1.0 {
		ratio = 1.0
	}

	// Calculate effective slippage: maxSlippageBps * ratio
	// Use integer arithmetic: (expectedOut * (10000 - effectiveSlippageBps)) / 10000
	effectiveSlippageBps := uint64(float64(maxSlippageBps) * ratio)
	if effectiveSlippageBps > 10000 {
		effectiveSlippageBps = 10000
	}

	// minOut = expectedOut * (10000 - effectiveSlippageBps) / 10000
	multiplier := uint64(10000 - effectiveSlippageBps)
	minOut := (int64(expectedOut) * int64(multiplier)) / 10000

	// Ensure we don't return 0 for valid expected outputs
	if minOut == 0 && expectedOut > 0 {
		minOut = 1
	}

	return minOut
}

// NewService creates a new router service
func NewService(config VSCConfig, dexExecutor DEXExecutor) *Service {
	adapters := NewChainAdapters()

	// Register built-in adapters
	// TODO: Load from configuration
	// ethAdapter := adapters.NewEthereumAdapter("0x...", "https://...")
	// adapters.RegisterAdapter(ethAdapter)

	return &Service{
		vscConfig:   config,
		adapters:    adapters,
		poolQuerier: nil, // Can be set via SetPoolQuerier
		dexExecutor: dexExecutor,
	}
}

// SetPoolQuerier sets the pool querier for the service
func (s *Service) SetPoolQuerier(querier PoolQuerier) {
	s.poolQuerier = querier
}

// ComputeRoute finds the optimal route for a swap (external API method)
func (s *Service) ComputeRoute(ctx context.Context, params SwapParams) (*SwapResult, error) {
	// Convert external params to internal format
	internalParams := SwapParams{
		Sender:         params.Sender,
		AmountIn:       params.AmountIn,
		AssetIn:        params.AssetIn,
		AssetOut:       params.AssetOut,
		MinAmountOut:   params.MinAmountOut,
		MaxSlippage:    params.MaxSlippage,
		MiddleOutRatio: params.MiddleOutRatio,
		Beneficiary:    params.Beneficiary,
		RefBps:         params.RefBps,
	}

	// ExecuteSwap never returns an error, only SwapResult with Success=false on failure
	result, _ := s.ExecuteSwap(internalParams)

	// Return full result including error message
	return result, nil
}

// ExecuteTransaction composes and submits the swap transaction
func (s *Service) ExecuteTransaction(ctx context.Context, result *SwapResult) error {
	log.Printf("Executing swap: %+v", result)

	// Execute the swap using the DEX executor
	if s.dexExecutor == nil {
		return fmt.Errorf("DEX executor not initialized")
	}

	// Execute the DEX swap via the executor
	err := s.dexExecutor.ExecuteDexSwap(ctx, result.AmountOut, result.Route, result.Fee0+result.Fee1)
	if err != nil {
		log.Printf("Failed to execute DEX swap: %v", err)
		return fmt.Errorf("failed to execute swap: %w", err)
	}

	log.Printf("DEX swap executed successfully - Amount Out: %d", result.AmountOut)
	return nil
}
