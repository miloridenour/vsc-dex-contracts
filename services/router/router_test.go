package router

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockDEXExecutor implements DEXExecutor for testing
type mockDEXExecutor struct{}

func (m *mockDEXExecutor) ExecuteDexSwap(ctx context.Context, amountOut int64, route []string, fee int64) error {
	// Mock implementation - just return success
	return nil
}

func TestNewService(t *testing.T) {
	config := VSCConfig{
		Endpoint: "http://localhost:4000",
		Key:      "test-key",
		Username: "test-user",
	}

	mockExecutor := &mockDEXExecutor{}
	svc := NewService(config, mockExecutor)

	assert.NotNil(t, svc)
	assert.NotNil(t, svc.adapters)
	assert.Equal(t, config, svc.vscConfig)
	assert.Equal(t, mockExecutor, svc.dexExecutor)
}

func TestComputeDirectRoute(t *testing.T) {
	mockExecutor := &mockDEXExecutor{}
	svc := NewService(VSCConfig{}, mockExecutor)

	// Test BTC -> HBD direct route (using placeholder pool logic)
	params := SwapParams{
		AssetIn:      "BTC",
		AssetOut:     "HBD",
		AmountIn:     100000, // 0.001 BTC
		MinAmountOut: 9000,   // Min 9 HBD
		MaxSlippage:  50,     // 0.5%
		Sender:       "test-user",
	}

	result, err := svc.ComputeRoute(context.Background(), params)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Greater(t, result.AmountOut, int64(0))
	assert.Equal(t, []string{"hive"}, result.Route) // Simplified route representation
	assert.True(t, result.Success)
}

func TestComputeHbdSavingsToHbdRoute(t *testing.T) {
	mockExecutor := &mockDEXExecutor{}
	svc := NewService(VSCConfig{}, mockExecutor)

	params := SwapParams{
		AssetIn:      "HBD_SAVINGS",
		AssetOut:     "HBD",
		AmountIn:     1000000, // 1 HBD_SAVINGS
		MinAmountOut: 900000,  // Min 0.90 HBD (allowing for fees and price impact)
		MaxSlippage:  50,
		Sender:       "test-user",
	}

	result, err := svc.ComputeRoute(context.Background(), params)
	require.NoError(t, err)
	require.NotNil(t, result)
	if !result.Success {
		t.Logf("Swap failed: %s", result.ErrorMessage)
	}
	assert.True(t, result.Success, "swap should succeed: %s", result.ErrorMessage)
	assert.Greater(t, result.AmountOut, int64(0))
	assert.Equal(t, []string{"hive"}, result.Route)
	// For 1:1 pool with fees, output should be slightly less than input
	assert.Less(t, result.AmountOut, params.AmountIn)
}

func TestComputeHiveToHbdRoute(t *testing.T) {
	mockExecutor := &mockDEXExecutor{}
	svc := NewService(VSCConfig{}, mockExecutor)

	params := SwapParams{
		AssetIn:      "HIVE",
		AssetOut:     "HBD",
		AmountIn:     1000000, // 1 HIVE
		MinAmountOut: 900000,  // Min 0.90 HBD (allowing for fees and price impact)
		MaxSlippage:  50,
		Sender:       "test-user",
	}

	result, err := svc.ComputeRoute(context.Background(), params)
	require.NoError(t, err)
	require.NotNil(t, result)
	if !result.Success {
		t.Logf("Swap failed: %s", result.ErrorMessage)
	}
	assert.True(t, result.Success, "swap should succeed: %s", result.ErrorMessage)
	assert.Greater(t, result.AmountOut, int64(0))
	assert.Equal(t, []string{"hive"}, result.Route)
}

func TestComputeBtcToHiveRoute(t *testing.T) {
	mockExecutor := &mockDEXExecutor{}
	svc := NewService(VSCConfig{}, mockExecutor)

	// BTC -> HIVE should route through HBD: BTC -> HBD -> HIVE
	params := SwapParams{
		AssetIn:      "BTC",
		AssetOut:     "HIVE",
		AmountIn:     100000, // 0.001 BTC
		MinAmountOut: 8000,   // Min 0.008 HIVE
		MaxSlippage:  50,
		Sender:       "test-user",
	}

	result, err := svc.ComputeRoute(context.Background(), params)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success, "swap should succeed")
	assert.Greater(t, result.AmountOut, int64(0))
	assert.Equal(t, 2, len(result.Route)) // Two-hop route through HBD
	assert.Contains(t, result.Route, "hive")
	assert.Contains(t, result.Route, "hbd-intermediate")
}

func TestUnsupportedRoute(t *testing.T) {
	mockExecutor := &mockDEXExecutor{}
	svc := NewService(VSCConfig{}, mockExecutor)

	// Test route that doesn't exist
	params := SwapParams{
		AssetIn:      "UNKNOWN",
		AssetOut:     "HBD",
		AmountIn:     1000000,
		MinAmountOut: 950000,
		MaxSlippage:  50,
		Sender:       "test-user",
	}

	result, err := svc.ComputeRoute(context.Background(), params)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.Success, "unsupported route should fail")
	assert.Contains(t, result.ErrorMessage, "no pool found")
}

func TestChainAdapters(t *testing.T) {
	adapters := NewChainAdapters()

	// Test empty state
	assert.Empty(t, adapters.ListChains())
	_, exists := adapters.GetAdapter("BTC")
	assert.False(t, exists)

	// Test adapter registration
	ethAdapter := &EthereumAdapter{
		ContractAddr: "0x123",
		RpcURL:       "https://mainnet.infura.io",
	}

	adapters.RegisterAdapter(ethAdapter)

	chains := adapters.ListChains()
	assert.Contains(t, chains, "ETH")

	adapter, exists := adapters.GetAdapter("ETH")
	assert.True(t, exists)
	assert.Equal(t, "ETH", adapter.Chain())
	assert.Equal(t, "WETH", adapter.GetMappedToken())
}

func TestEthereumAdapter(t *testing.T) {
	adapter := &EthereumAdapter{
		ContractAddr: "0x123",
		RpcURL:       "https://mainnet.infura.io",
	}

	assert.Equal(t, "ETH", adapter.Chain())
	assert.Equal(t, "WETH", adapter.GetMappedToken())
	assert.Equal(t, uint32(12), adapter.GetRequiredConfirmations())
	assert.Equal(t, "0x123", adapter.GetContractAddress())

	// Test address validation
	valid, err := adapter.FormatAddress("0x742d35Cc6537C0532915a3d6446c5814c100d420")
	require.NoError(t, err)
	assert.Equal(t, "0x742d35Cc6537C0532915a3d6446c5814c100d420", valid)

	// Test invalid address
	_, err = adapter.FormatAddress("invalid-address")
	assert.Error(t, err)
}

func TestSolanaAdapter(t *testing.T) {
	adapter := &SolanaAdapter{
		ContractAddr: "solana-program-id",
		RpcURL:       "https://api.mainnet.solana.com",
	}

	assert.Equal(t, "SOL", adapter.Chain())
	assert.Equal(t, "WSOL", adapter.GetMappedToken())
	assert.Equal(t, uint32(32), adapter.GetRequiredConfirmations())

	// Test address validation (basic)
	valid, err := adapter.FormatAddress("11111111111111111111111111111112")
	require.NoError(t, err)
	assert.Equal(t, "11111111111111111111111111111112", valid)

	// Test invalid address (too short)
	_, err = adapter.FormatAddress("short")
	assert.Error(t, err)
}

// Edge case tests

func TestSwapSameAsset(t *testing.T) {
	mockExecutor := &mockDEXExecutor{}
	svc := NewService(VSCConfig{}, mockExecutor)

	params := SwapParams{
		AssetIn:      "BTC",
		AssetOut:     "BTC",
		AmountIn:     100000,
		MinAmountOut: 90000,
		MaxSlippage:  50,
		Sender:       "test-user",
	}

	result, err := svc.ComputeRoute(context.Background(), params)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.Success, "swapping same asset should fail")
	assert.Contains(t, result.ErrorMessage, "cannot swap asset to itself")
}

func TestSwapZeroAmount(t *testing.T) {
	mockExecutor := &mockDEXExecutor{}
	svc := NewService(VSCConfig{}, mockExecutor)

	params := SwapParams{
		AssetIn:      "BTC",
		AssetOut:     "HBD",
		AmountIn:     0,
		MinAmountOut: 0,
		MaxSlippage:  50,
		Sender:       "test-user",
	}

	result, err := svc.ComputeRoute(context.Background(), params)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.Success, "zero amount swap should fail")
	assert.Contains(t, result.ErrorMessage, "must be greater than zero")
}

func TestSwapNegativeAmount(t *testing.T) {
	mockExecutor := &mockDEXExecutor{}
	svc := NewService(VSCConfig{}, mockExecutor)

	params := SwapParams{
		AssetIn:      "BTC",
		AssetOut:     "HBD",
		AmountIn:     -100000,
		MinAmountOut: 0,
		MaxSlippage:  50,
		Sender:       "test-user",
	}

	result, err := svc.ComputeRoute(context.Background(), params)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.Success, "negative amount swap should fail")
}

func TestSwapSlippageExceeded(t *testing.T) {
	mockExecutor := &mockDEXExecutor{}
	svc := NewService(VSCConfig{}, mockExecutor)

	// Set an unrealistically high minimum output
	params := SwapParams{
		AssetIn:      "BTC",
		AssetOut:     "HBD",
		AmountIn:     100000, // 0.001 BTC
		MinAmountOut: 999999999, // Unrealistically high
		MaxSlippage:  50,
		Sender:       "test-user",
	}

	result, err := svc.ComputeRoute(context.Background(), params)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.Success, "swap should fail due to slippage")
	assert.Contains(t, result.ErrorMessage, "slippage")
}

func TestSwapInvalidPool(t *testing.T) {
	mockExecutor := &mockDEXExecutor{}
	svc := NewService(VSCConfig{}, mockExecutor)

	params := SwapParams{
		AssetIn:      "UNKNOWN",
		AssetOut:     "HBD",
		AmountIn:     1000000,
		MinAmountOut: 950000,
		MaxSlippage:  50,
		Sender:       "test-user",
	}

	result, err := svc.ComputeRoute(context.Background(), params)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.Success, "invalid pool should fail")
	assert.Contains(t, result.ErrorMessage, "no pool found")
}

func TestBtcToHbdSwap(t *testing.T) {
	mockExecutor := &mockDEXExecutor{}
	svc := NewService(VSCConfig{}, mockExecutor)

	params := SwapParams{
		AssetIn:      "BTC",
		AssetOut:     "HBD",
		AmountIn:     100000, // 0.001 BTC
		MinAmountOut: 0,      // No minimum
		MaxSlippage:  100,    // 1% slippage
		Sender:       "test-user",
	}

	result, err := svc.ComputeRoute(context.Background(), params)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success, "BTC->HBD swap should succeed")
	assert.Greater(t, result.AmountOut, int64(0))
	// HBD amount should match output when swapping to HBD
	assert.Equal(t, result.HbdAmount, result.AmountOut, "HBD amount should match output for HBD swap")
	assert.Greater(t, result.AmountOut, int64(0))
}

func TestHbdToBtcSwap(t *testing.T) {
	mockExecutor := &mockDEXExecutor{}
	svc := NewService(VSCConfig{}, mockExecutor)

	params := SwapParams{
		AssetIn:      "HBD",
		AssetOut:     "BTC",
		AmountIn:     1000000, // 1 HBD
		MinAmountOut: 0,
		MaxSlippage:  100,
		Sender:       "test-user",
	}

	result, err := svc.ComputeRoute(context.Background(), params)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success, "HBD->BTC swap should succeed")
	assert.Greater(t, result.AmountOut, int64(0))
	// HBD amount should match input when swapping from HBD
	assert.Equal(t, result.HbdAmount, params.AmountIn, "HBD amount should match input for HBD->BTC swap")
	assert.Greater(t, result.AmountOut, int64(0))
}

func TestHbdSavingsToHbdSwap(t *testing.T) {
	mockExecutor := &mockDEXExecutor{}
	svc := NewService(VSCConfig{}, mockExecutor)

	params := SwapParams{
		AssetIn:      "HBD_SAVINGS",
		AssetOut:     "HBD",
		AmountIn:     1000000, // 1 HBD_SAVINGS
		MinAmountOut: 0,       // No minimum - allow fees
		MaxSlippage:  100,
		Sender:       "test-user",
	}

	result, err := svc.ComputeRoute(context.Background(), params)
	require.NoError(t, err)
	require.NotNil(t, result)
	if !result.Success {
		t.Logf("Swap failed: %s", result.ErrorMessage)
	}
	assert.True(t, result.Success, "HBD_SAVINGS->HBD swap should succeed")
	assert.Greater(t, result.AmountOut, int64(0))
	assert.Equal(t, result.HbdAmount, result.AmountOut)
	// With fees, output should be slightly less than input
	assert.Less(t, result.AmountOut, params.AmountIn)
}

func TestHiveToHbdSwap(t *testing.T) {
	mockExecutor := &mockDEXExecutor{}
	svc := NewService(VSCConfig{}, mockExecutor)

	params := SwapParams{
		AssetIn:      "HIVE",
		AssetOut:     "HBD",
		AmountIn:     1000000, // 1 HIVE
		MinAmountOut: 0,       // No minimum - allow fees
		MaxSlippage:  100,
		Sender:       "test-user",
	}

	result, err := svc.ComputeRoute(context.Background(), params)
	require.NoError(t, err)
	require.NotNil(t, result)
	if !result.Success {
		t.Logf("Swap failed: %s", result.ErrorMessage)
	}
	assert.True(t, result.Success, "HIVE->HBD swap should succeed")
	assert.Greater(t, result.AmountOut, int64(0))
	assert.Equal(t, result.HbdAmount, result.AmountOut)
}

func TestBtcToHiveTwoHopSwap(t *testing.T) {
	mockExecutor := &mockDEXExecutor{}
	svc := NewService(VSCConfig{}, mockExecutor)

	params := SwapParams{
		AssetIn:      "BTC",
		AssetOut:     "HIVE",
		AmountIn:     100000, // 0.001 BTC
		MinAmountOut: 0,
		MaxSlippage:  100,
		Sender:       "test-user",
	}

	result, err := svc.ComputeRoute(context.Background(), params)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success, "BTC->HIVE two-hop swap should succeed")
	assert.Greater(t, result.AmountOut, int64(0))
	assert.Equal(t, 2, len(result.Route), "should be a two-hop route")
	assert.Greater(t, result.HbdAmount, int64(0), "should have intermediate HBD amount")
}

func TestHiveToBtcTwoHopSwap(t *testing.T) {
	mockExecutor := &mockDEXExecutor{}
	svc := NewService(VSCConfig{}, mockExecutor)

	params := SwapParams{
		AssetIn:      "HIVE",
		AssetOut:     "BTC",
		AmountIn:     1000000, // 1 HIVE
		MinAmountOut: 0,
		MaxSlippage:  100,
		Sender:       "test-user",
	}

	result, err := svc.ComputeRoute(context.Background(), params)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success, "HIVE->BTC two-hop swap should succeed")
	assert.Greater(t, result.AmountOut, int64(0))
	assert.Equal(t, 2, len(result.Route), "should be a two-hop route")
}

func TestHbdSavingsToHiveTwoHopSwap(t *testing.T) {
	mockExecutor := &mockDEXExecutor{}
	svc := NewService(VSCConfig{}, mockExecutor)

	params := SwapParams{
		AssetIn:      "HBD_SAVINGS",
		AssetOut:     "HIVE",
		AmountIn:     1000000, // 1 HBD_SAVINGS
		MinAmountOut: 0,
		MaxSlippage:  100,
		Sender:       "test-user",
	}

	result, err := svc.ComputeRoute(context.Background(), params)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success, "HBD_SAVINGS->HIVE two-hop swap should succeed")
	assert.Greater(t, result.AmountOut, int64(0))
	assert.Equal(t, 2, len(result.Route), "should be a two-hop route")
}

func TestCalculateExpectedOutput(t *testing.T) {
	mockExecutor := &mockDEXExecutor{}
	svc := NewService(VSCConfig{}, mockExecutor)

	// Test with known pool reserves (1 BTC = 10 HBD from defaults)
	amountIn := int64(100000) // 0.001 BTC
	expectedOut, err := svc.calculateExpectedOutput(amountIn, "btc-hbd-pool", "BTC", "HBD")
	require.NoError(t, err)
	assert.Greater(t, expectedOut, int64(0))
	
	// With 1 BTC reserve and 10 HBD reserve, 0.001 BTC should give roughly 0.01 HBD
	// But after fees, it should be slightly less
	// Exact calculation: k = 100000000 * 10000000 = 1e15
	// amountInAfterFee = 100000 * 9992 / 10000 = 99920
	// newReserveIn = 100000000 + 99920 = 100099920
	// amountOut = 10000000 - (1e15 / 100099920) ≈ 10000000 - 9990002 ≈ 9998
	// So we expect around 9998 HBD (0.009998 HBD)
	assert.Less(t, expectedOut, int64(10000), "output should be less than 0.01 HBD")
	assert.Greater(t, expectedOut, int64(9900), "output should be at least 0.0099 HBD")
}

func TestCalculateMinOutput(t *testing.T) {
	mockExecutor := &mockDEXExecutor{}
	svc := NewService(VSCConfig{}, mockExecutor)

	expectedOut := int64(1000000)
	maxSlippage := uint64(50) // 0.5%
	ratio := 1.0

	minOut := svc.calculateMinOutput(expectedOut, maxSlippage, ratio)
	
	// With 0.5% slippage, min should be 99.5% of expected
	expectedMin := int64(float64(expectedOut) * 0.995)
	assert.Equal(t, expectedMin, minOut)
}

func TestCalculateMinOutputWithRatio(t *testing.T) {
	mockExecutor := &mockDEXExecutor{}
	svc := NewService(VSCConfig{}, mockExecutor)

	expectedOut := int64(1000000)
	maxSlippage := uint64(100) // 1%
	ratio := 0.5 // Use 50% of slippage tolerance

	minOut := svc.calculateMinOutput(expectedOut, maxSlippage, ratio)
	
	// With 1% slippage and 0.5 ratio, min should be 99.5% of expected
	// Using integer arithmetic: 1000000 * 9950 / 10000 = 995000
	expectedMin := int64(995000)
	assert.Equal(t, expectedMin, minOut)
}

// Edge case tests for overflow and pool protection

func TestPoolDrainProtection(t *testing.T) {
	mockExecutor := &mockDEXExecutor{}
	svc := NewService(VSCConfig{}, mockExecutor)

	// Try to swap more than 50% of reserve
	params := SwapParams{
		AssetIn:      "BTC",
		AssetOut:     "HBD",
		AmountIn:     60000000, // 0.6 BTC (60% of 1 BTC reserve)
		MinAmountOut: 0,
		MaxSlippage:  100,
		Sender:       "test-user",
	}

	result, err := svc.ComputeRoute(context.Background(), params)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.Success, "swap should fail due to pool drain protection")
	assert.Contains(t, result.ErrorMessage, "exceeds maximum allowed")
	assert.Contains(t, result.ErrorMessage, "50%")
}

func TestSwapAmountTooSmallAfterFees(t *testing.T) {
	mockExecutor := &mockDEXExecutor{}
	svc := NewService(VSCConfig{}, mockExecutor)

	// Try to swap amount that would be 0 after fees
	params := SwapParams{
		AssetIn:      "BTC",
		AssetOut:     "HBD",
		AmountIn:     1, // 1 satoshi - very small
		MinAmountOut: 0,
		MaxSlippage:  100,
		Sender:       "test-user",
	}

	result, err := svc.ComputeRoute(context.Background(), params)
	require.NoError(t, err)
	require.NotNil(t, result)
	// Should either succeed with very small output or fail with "too small after fees"
	// The current implementation should reject amounts that result in 0 after fee
	if !result.Success {
		assert.Contains(t, result.ErrorMessage, "too small")
	}
}

func TestInvalidFee(t *testing.T) {
	mockExecutor := &mockDEXExecutor{}
	svc := NewService(VSCConfig{}, mockExecutor)

	// Create a mock pool querier with invalid fee
	mockQuerier := &MockPoolQuerier{
		pools: map[string]*PoolInfoWithReserves{
			"btc-hbd-pool": {
				ContractId: "btc-hbd-pool",
				Asset0:     "BTC",
				Asset1:     "HBD",
				Reserve0:   100000000,
				Reserve1:   10000000,
				Fee:        20000, // 200% fee - invalid
			},
		},
	}
	svc.SetPoolQuerier(mockQuerier)

	params := SwapParams{
		AssetIn:      "BTC",
		AssetOut:     "HBD",
		AmountIn:     100000,
		MinAmountOut: 0,
		MaxSlippage:  100,
		Sender:       "test-user",
	}

	result, err := svc.ComputeRoute(context.Background(), params)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.Success, "swap should fail with invalid fee")
	assert.Contains(t, result.ErrorMessage, "invalid fee")
}

func TestLargeReserveCalculation(t *testing.T) {
	mockExecutor := &mockDEXExecutor{}
	svc := NewService(VSCConfig{}, mockExecutor)

	// Create a pool with very large reserves to test overflow protection
	mockQuerier := &MockPoolQuerier{
		pools: map[string]*PoolInfoWithReserves{
			"btc-hbd-pool": {
				ContractId: "btc-hbd-pool",
				Asset0:     "BTC",
				Asset1:     "HBD",
				Reserve0:   1000000000000000, // 10M BTC in satoshis
				Reserve1:   100000000000000,  // 100M HBD
				Fee:        8,
			},
		},
	}
	svc.SetPoolQuerier(mockQuerier)

	params := SwapParams{
		AssetIn:      "BTC",
		AssetOut:     "HBD",
		AmountIn:     100000000, // 1 BTC
		MinAmountOut: 0,
		MaxSlippage:  100,
		Sender:       "test-user",
	}

	result, err := svc.ComputeRoute(context.Background(), params)
	require.NoError(t, err)
	require.NotNil(t, result)
	// Should succeed with big.Int calculations preventing overflow
	assert.True(t, result.Success, "swap should succeed even with large reserves")
	assert.Greater(t, result.AmountOut, int64(0))
}

func TestTwoHopSwapFirstFails(t *testing.T) {
	mockExecutor := &mockDEXExecutor{}
	svc := NewService(VSCConfig{}, mockExecutor)

	// Create a scenario where first swap would fail
	// This tests error handling in two-hop swaps
	params := SwapParams{
		AssetIn:      "BTC",
		AssetOut:     "HIVE",
		AmountIn:     1, // Very small amount that might fail
		MinAmountOut: 999999999, // Unrealistically high minimum
		MaxSlippage:  50,
		Sender:       "test-user",
	}

	result, err := svc.ComputeRoute(context.Background(), params)
	require.NoError(t, err)
	require.NotNil(t, result)
	// Should fail and return empty route
	assert.False(t, result.Success)
	assert.Equal(t, []string{}, result.Route, "route should be empty on failure")
}

// MockPoolQuerier for testing
type MockPoolQuerier struct {
	pools map[string]*PoolInfoWithReserves
}

func (m *MockPoolQuerier) GetPoolByID(poolID string) (*PoolInfoWithReserves, error) {
	pool, ok := m.pools[poolID]
	if !ok {
		return nil, fmt.Errorf("pool not found: %s", poolID)
	}
	return pool, nil
}

func (m *MockPoolQuerier) GetPoolsByAsset(asset string) ([]PoolInfoWithReserves, error) {
	var pools []PoolInfoWithReserves
	for _, pool := range m.pools {
		if pool.Asset0 == asset || pool.Asset1 == asset {
			pools = append(pools, *pool)
		}
	}
	return pools, nil
}
