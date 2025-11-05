package router

import (
	"context"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewService(t *testing.T) {
	config := VSCConfig{
		Endpoint: "http://localhost:4000",
		Key:      "test-key",
		Username: "test-user",
	}

	svc := NewService(config)

	assert.NotNil(t, svc)
	assert.NotNil(t, svc.adapters)
	assert.Equal(t, config, svc.vscConfig)
}

func TestComputeDirectRoute(t *testing.T) {
	svc := NewService(VSCConfig{})

	// Test BTC -> HBD direct route (now supported)
	params := SwapParams{
		FromAsset:   "BTC",
		ToAsset:     "HBD",
		Amount:      100000, // 0.001 BTC
		MinOut:      9000,   // Min 9 HBD
		SlippageBps: 50,     // 0.5%
		Sender:      "test-user",
	}

	result, err := svc.ComputeRoute(context.Background(), params)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Greater(t, result.AmountOut, int64(0))
	assert.Equal(t, 1, len(result.Route)) // Direct route has 1 hop
}

func TestComputeHbdSavingsToHbdRoute(t *testing.T) {
	svc := NewService(VSCConfig{})

	params := SwapParams{
		FromAsset:   "HBD_SAVINGS",
		ToAsset:     "HBD",
		Amount:      1000000, // 1 HBD_SAVINGS
		MinOut:      950000,  // Min 0.95 HBD
		SlippageBps: 50,
		Sender:      "test-user",
	}

	result, err := svc.ComputeRoute(context.Background(), params)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Greater(t, result.AmountOut, int64(0))
}

func TestComputeHiveToHbdRoute(t *testing.T) {
	svc := NewService(VSCConfig{})

	params := SwapParams{
		FromAsset:   "HIVE",
		ToAsset:     "HBD",
		Amount:      1000000, // 1 HIVE
		MinOut:      950000,  // Min 0.95 HBD
		SlippageBps: 50,
		Sender:      "test-user",
	}

	result, err := svc.ComputeRoute(context.Background(), params)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Greater(t, result.AmountOut, int64(0))
}

func TestComputeBtcToHiveRoute(t *testing.T) {
	svc := NewService(VSCConfig{})

	// BTC -> HIVE should route through HBD: BTC -> HBD -> HIVE
	params := SwapParams{
		FromAsset:   "BTC",
		ToAsset:     "HIVE",
		Amount:      100000, // 0.001 BTC
		MinOut:      8000,   // Min 0.008 HIVE
		SlippageBps: 50,
		Sender:      "test-user",
	}

	result, err := svc.ComputeRoute(context.Background(), params)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Greater(t, result.AmountOut, int64(0))
	assert.Equal(t, 2, len(result.Route)) // Two-hop route through HBD
}

func TestUnsupportedRoute(t *testing.T) {
	svc := NewService(VSCConfig{})

	// Test route that doesn't exist
	params := SwapParams{
		FromAsset:   "UNKNOWN",
		ToAsset:     "HBD",
		Amount:      1000000,
		MinOut:      950000,
		SlippageBps: 50,
		Sender:      "test-user",
	}

	_, err := svc.ComputeRoute(context.Background(), params)
	assert.Error(t, err)
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

func TestCalculateOutput(t *testing.T) {
	svc := NewService(VSCConfig{})

	// Test AMM calculation: (x + dx) * (y - dy) = x * y
	// dy = (y * dx) / (x + dx)
	reserveIn := big.NewInt(1000000)   // 1 BTC reserve
	reserveOut := big.NewInt(10000000) // 10 HBD reserve
	amountIn := int64(100000)          // 0.1 BTC in

	output := svc.calculateOutput(amountIn, reserveIn, reserveOut)

	// Expected output: (10000000 * 100000) / (1000000 + 100000) = ~909090
	expected := int64(909090)
	assert.Equal(t, expected, output)
}
