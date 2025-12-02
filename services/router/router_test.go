package router

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockDEXExecutor implements DEXExecutor for testing
type mockDEXExecutor struct {
	executedOperations []string // Track executed operations for testing
}

func (m *mockDEXExecutor) ExecuteDexOperation(ctx context.Context, operationType string, payload string) error {
	// Track the operation for testing
	m.executedOperations = append(m.executedOperations, operationType+":"+payload)
	return nil
}

func (m *mockDEXExecutor) ExecuteDexSwap(ctx context.Context, amountOut int64, route []string, fee int64) error {
	// Keep for backward compatibility
	return nil
}

func TestNewService(t *testing.T) {
	config := VSCConfig{
		Endpoint:          "http://localhost:4000",
		Key:               "test-key",
		Username:          "test-user",
		DexRouterContract: "dex-router-contract",
	}

	mockExecutor := &mockDEXExecutor{}
	svc := NewService(config, mockExecutor)

	assert.NotNil(t, svc)
	assert.Equal(t, config, svc.vscConfig)
	assert.Equal(t, mockExecutor, svc.dexExecutor)
}

func TestExecuteSwap(t *testing.T) {
	mockExecutor := &mockDEXExecutor{}
	config := VSCConfig{DexRouterContract: "dex-router-contract"}
	svc := NewService(config, mockExecutor)

	// Test HBD -> HIVE swap
	params := SwapParams{
		AssetIn:      "HBD",
		AssetOut:     "HIVE",
		AmountIn:     1000000, // 1 HBD
		MinAmountOut: 900000,  // Min 0.9 HIVE
		MaxSlippage:  50,      // 0.5%
		Sender:       "test-user",
		Beneficiary:  "ref-user",
		RefBps:       25, // 0.25% referral
	}

	result, err := svc.ExecuteSwap(params)

	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, []string{"direct"}, result.Route)

	// Verify the JSON payload was constructed correctly
	require.Len(t, mockExecutor.executedOperations, 1)
	operation := mockExecutor.executedOperations[0]

	// Should start with "execute:"
	assert.True(t, strings.HasPrefix(operation, "execute:"))

	// Extract and parse the JSON payload
	payload := strings.TrimPrefix(operation, "execute:")
	var instruction map[string]interface{}
	err = json.Unmarshal([]byte(payload), &instruction)
	require.NoError(t, err)

	// Verify the instruction structure
	assert.Equal(t, "swap", instruction["type"])
	assert.Equal(t, "1.0.0", instruction["version"])
	assert.Equal(t, "HBD", instruction["asset_in"])
	assert.Equal(t, "HIVE", instruction["asset_out"])
	assert.Equal(t, "test-user", instruction["recipient"])
	assert.Equal(t, float64(900000), instruction["min_amount_out"])
	assert.Equal(t, float64(50), instruction["slippage_bps"])
	assert.Equal(t, "ref-user", instruction["beneficiary"])
	assert.Equal(t, float64(25), instruction["ref_bps"])
}

func TestExecuteDeposit(t *testing.T) {
	mockExecutor := &mockDEXExecutor{}
	config := VSCConfig{DexRouterContract: "dex-router-contract"}
	svc := NewService(config, mockExecutor)

	// Test liquidity deposit
	params := DepositParams{
		AssetIn:  "HBD",
		AssetOut: "HIVE",
		AmountIn: 1000000,
		Sender:   "test-user",
	}

	result, err := svc.ExecuteDeposit(params)

	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, []string{"deposit"}, result.Route)

	// Verify the JSON payload
	require.Len(t, mockExecutor.executedOperations, 1)
	operation := mockExecutor.executedOperations[0]
	assert.True(t, strings.HasPrefix(operation, "execute:"))

	payload := strings.TrimPrefix(operation, "execute:")
	var instruction map[string]interface{}
	err = json.Unmarshal([]byte(payload), &instruction)
	require.NoError(t, err)

	assert.Equal(t, "deposit", instruction["type"])
	assert.Equal(t, "1.0.0", instruction["version"])
	assert.Equal(t, "HBD", instruction["asset_in"])
	assert.Equal(t, "HIVE", instruction["asset_out"])
	assert.Equal(t, "test-user", instruction["recipient"])
}

func TestExecuteWithdrawal(t *testing.T) {
	mockExecutor := &mockDEXExecutor{}
	config := VSCConfig{DexRouterContract: "dex-router-contract"}
	svc := NewService(config, mockExecutor)

	// Test liquidity withdrawal
	params := WithdrawalParams{
		AssetIn:  "HBD",
		AssetOut: "HIVE",
		LpAmount: 100000,
		Sender:   "test-user",
	}

	result, err := svc.ExecuteWithdrawal(params)

	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, []string{"withdrawal"}, result.Route)

	// Verify the JSON payload
	require.Len(t, mockExecutor.executedOperations, 1)
	operation := mockExecutor.executedOperations[0]
	assert.True(t, strings.HasPrefix(operation, "execute:"))

	payload := strings.TrimPrefix(operation, "execute:")
	var instruction map[string]interface{}
	err = json.Unmarshal([]byte(payload), &instruction)
	require.NoError(t, err)

	assert.Equal(t, "withdrawal", instruction["type"])
	assert.Equal(t, "1.0.0", instruction["version"])
	assert.Equal(t, "HBD", instruction["asset_in"])
	assert.Equal(t, "HIVE", instruction["asset_out"])
	assert.Equal(t, "test-user", instruction["recipient"])
}

func TestSwapValidation(t *testing.T) {
	mockExecutor := &mockDEXExecutor{}
	config := VSCConfig{DexRouterContract: "dex-router-contract"}
	svc := NewService(config, mockExecutor)

	// Test same asset swap (should fail)
	params := SwapParams{
		AssetIn:      "HBD",
		AssetOut:     "HBD", // Same as input
		AmountIn:     1000000,
		MinAmountOut: 900000,
		Sender:       "test-user",
	}

	result, err := svc.ExecuteSwap(params)
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.ErrorMessage, "cannot swap asset to itself")
}

func TestServiceCreation(t *testing.T) {
	config := VSCConfig{
		Endpoint:          "http://localhost:4000",
		Key:               "test-key",
		Username:          "test-user",
		DexRouterContract: "dex-router-contract",
	}

	mockExecutor := &mockDEXExecutor{}
	svc := NewService(config, mockExecutor)

	assert.NotNil(t, svc)
	assert.Equal(t, config, svc.vscConfig)
	assert.Equal(t, mockExecutor, svc.dexExecutor)
}
