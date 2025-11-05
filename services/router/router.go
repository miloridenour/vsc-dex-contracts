package router

import (
	"context"
	"fmt"
	"math/big"
)

// Service provides DEX routing and transaction composition
type Service struct {
	vscConfig VSCConfig
	adapters  *ChainAdapters
}

type VSCConfig struct {
	Endpoint string
	Key      string
	Username string
}

// SwapParams represents a swap request
type SwapParams struct {
	FromAsset   string
	ToAsset     string
	Amount      int64
	MinOut      int64
	SlippageBps uint64
	Sender      string
}

// SwapResult represents the result of a route computation
type SwapResult struct {
	Route      []PoolHop
	AmountOut  int64
	PriceImpact float64
	Fee         int64
	TxPayload   []byte
}

// PoolHop represents a hop in a swap route
type PoolHop struct {
	PoolID     string
	TokenIn    string
	TokenOut   string
	AmountIn   int64
	AmountOut  int64
}

// NewService creates a new router service
func NewService(config VSCConfig) *Service {
	adapters := NewChainAdapters()

	// Register built-in adapters
	// TODO: Load from configuration
	// ethAdapter := adapters.NewEthereumAdapter("0x...", "https://...")
	// adapters.RegisterAdapter(ethAdapter)

	return &Service{
		vscConfig: config,
		adapters:  adapters,
	}
}

// ComputeRoute finds the optimal route for a swap
func (s *Service) ComputeRoute(ctx context.Context, params SwapParams) (*SwapResult, error) {
	// Check if this is a direct pair
	if directPool, exists := s.findDirectPool(params.FromAsset, params.ToAsset); exists {
		return s.computeDirectRoute(params, directPool)
	}

	// Check for two-hop routes through HBD (like original DEX)
	if hbdRoute := s.findHBDRoute(params); hbdRoute != nil {
		return hbdRoute, nil
	}

	// Check for multi-hop routes
	return s.findMultiHopRoute(params)
}

// computeDirectRoute computes a direct swap route
func (s *Service) computeDirectRoute(params SwapParams, poolID string) (*SwapResult, error) {
	reserves := s.getPoolReserves(poolID)
	if reserves == nil {
		return nil, fmt.Errorf("could not get reserves for pool %s", poolID)
	}

	// Calculate output using AMM formula
	amountOut := s.calculateOutput(params.Amount, reserves.In, reserves.Out)

	// Apply slippage protection
	minOut := (amountOut * int64(10000-params.SlippageBps)) / 10000
	if minOut < params.MinOut {
		minOut = params.MinOut
	}

	// Estimate fee (0.3% like Uniswap)
	fee := amountOut / 333 // ~0.3%

	result := &SwapResult{
		Route: []PoolHop{{
			PoolID:    poolID,
			TokenIn:   params.FromAsset,
			TokenOut:  params.ToAsset,
			AmountIn:  params.Amount,
			AmountOut: amountOut,
		}},
		AmountOut:  amountOut,
		PriceImpact: s.calculatePriceImpact(params.Amount, reserves.In, reserves.Out),
		Fee:        fee,
	}

	return result, nil
}

// findHBDRoute finds a two-hop route through HBD
func (s *Service) findHBDRoute(params SwapParams) *SwapResult {
	// First hop: FromAsset -> HBD
	firstPool, firstExists := s.findDirectPool(params.FromAsset, "HBD")
	if !firstExists {
		return nil
	}

	// Second hop: HBD -> ToAsset
	secondPool, secondExists := s.findDirectPool("HBD", params.ToAsset)
	if !secondExists {
		return nil
	}

	firstReserves := s.getPoolReserves(firstPool)
	secondReserves := s.getPoolReserves(secondPool)

	if firstReserves == nil || secondReserves == nil {
		return nil
	}

	// Calculate amounts
	hbdOut := s.calculateOutput(params.Amount, firstReserves.In, firstReserves.Out)
	finalOut := s.calculateOutput(hbdOut, secondReserves.In, secondReserves.Out)

	route := []PoolHop{
		{
			PoolID:    firstPool,
			TokenIn:   params.FromAsset,
			TokenOut:  "HBD",
			AmountIn:  params.Amount,
			AmountOut: hbdOut,
		},
		{
			PoolID:    secondPool,
			TokenIn:   "HBD",
			TokenOut:  params.ToAsset,
			AmountIn:  hbdOut,
			AmountOut: finalOut,
		},
	}

	return &SwapResult{
		Route:      route,
		AmountOut:  finalOut,
		PriceImpact: 0, // TODO: calculate combined impact
		Fee:        (hbdOut + finalOut) / 333, // Combined fees
	}
}

// findMultiHopRoute finds complex multi-hop routes
func (s *Service) findMultiHopRoute(params SwapParams) (*SwapResult, error) {
	// TODO: Implement multi-hop route finding algorithm
	// For now, return nil
	return nil, fmt.Errorf("multi-hop routes not implemented")
}

// Helper functions (simplified - would query indexer in real implementation)

func (s *Service) findDirectPool(assetA, assetB string) (string, bool) {
	// TODO: Query indexer for pools containing both assets
	// Placeholder implementation with basic pool support

	// Ensure consistent ordering
	if assetA > assetB {
		assetA, assetB = assetB, assetA
	}

	switch {
	case (assetA == "BTC" && assetB == "HBD") || (assetA == "HBD" && assetB == "BTC"):
		return "btc-hbd-pool", true
	case (assetA == "HBD_SAVINGS" && assetB == "HBD") || (assetA == "HBD" && assetB == "HBD_SAVINGS"):
		return "hbd-savings-hbd-pool", true
	case (assetA == "HBD" && assetB == "HIVE") || (assetA == "HIVE" && assetB == "HBD"):
		return "hive-hbd-pool", true
	}

	return "", false
}

type Reserves struct {
	In  *big.Int
	Out *big.Int
}

func (s *Service) getPoolReserves(poolID string) *Reserves {
	// TODO: Query indexer for current pool reserves
	// Placeholder implementation
	switch poolID {
	case "btc-hbd-pool":
		return &Reserves{
			In:  big.NewInt(1000000), // 1 BTC worth of reserves
			Out: big.NewInt(10000000), // 10 HBD worth of reserves
		}
	case "hbd-savings-hbd-pool":
		return &Reserves{
			In:  big.NewInt(10000000), // 10 HBD_SAVINGS worth of reserves
			Out: big.NewInt(10000000), // 10 HBD worth of reserves
		}
	case "hive-hbd-pool":
		return &Reserves{
			In:  big.NewInt(10000000), // 10 HIVE worth of reserves
			Out: big.NewInt(10000000), // 10 HBD worth of reserves
		}
	}
	return nil
}

func (s *Service) calculateOutput(amountIn int64, reserveIn, reserveOut *big.Int) int64 {
	// AMM constant product formula: (x + dx) * (y - dy) = x * y
	// dy = (y * dx) / (x + dx)
	x := big.NewInt(reserveIn.Int64())
	y := big.NewInt(reserveOut.Int64())
	dx := big.NewInt(amountIn)

	// dy = (y * dx) / (x + dx)
	num := new(big.Int).Mul(y, dx)
	denom := new(big.Int).Add(x, dx)
	dy := new(big.Int).Div(num, denom)

	return dy.Int64()
}

func (s *Service) calculatePriceImpact(amountIn int64, reserveIn, reserveOut *big.Int) float64 {
	// Simplified price impact calculation
	// Impact = (actual price - spot price) / spot price
	spotPrice := float64(reserveOut.Int64()) / float64(reserveIn.Int64())
	actualPrice := float64(s.calculateOutput(amountIn, reserveIn, reserveOut)) / float64(amountIn)

	if spotPrice == 0 {
		return 0
	}

	return (actualPrice - spotPrice) / spotPrice
}

// ExecuteSwap composes and submits the swap transaction
func (s *Service) ExecuteSwap(ctx context.Context, result *SwapResult) error {
	// TODO: Compose contract call transaction
	// TODO: Sign and broadcast via VSC

	return fmt.Errorf("swap execution not implemented")
}
