package e2e

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestFullBtcHbdFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	// Setup test environment
	testEnv := setupTestEnvironment(t, ctx)
	defer testEnv.cleanup()

	t.Run("DeployContracts", func(t *testing.T) {
		err := deployContracts(t, ctx)
		require.NoError(t, err, "Contract deployment failed")
	})

	t.Run("RegisterBtcToken", func(t *testing.T) {
		err := registerBtcToken(t, ctx, testEnv)
		require.NoError(t, err, "BTC token registration failed")
	})

	t.Run("SubmitBtcHeaders", func(t *testing.T) {
		err := submitBtcHeaders(t, ctx, testEnv)
		require.NoError(t, err, "BTC header submission failed")
	})

	t.Run("ProveDeposit", func(t *testing.T) {
		depositTx, err := createTestDeposit(t, ctx)
		require.NoError(t, err, "Test deposit creation failed")

		proof := generateSPVProof(t, depositTx.txid, depositTx.vout)
		require.NotEmpty(t, proof, "SPV proof generation failed")

		mintedAmount, err := proveDeposit(t, ctx, testEnv, proof)
		require.NoError(t, err, "Deposit proof submission failed")
		require.Greater(t, mintedAmount, uint64(0), "No tokens minted")
	})

	t.Run("CreateBtcHbdPool", func(t *testing.T) {
		poolID, err := createBtcHbdPool(t, ctx, testEnv)
		require.NoError(t, err, "BTC/HBD pool creation failed")
		require.NotEmpty(t, poolID, "Pool ID should not be empty")

		testEnv.poolID = poolID
	})

	t.Run("AddLiquidity", func(t *testing.T) {
		err := addLiquidityToPool(t, ctx, testEnv, testEnv.poolID, 1000000, 10000000) // 1 BTC + 10 HBD
		require.NoError(t, err, "Adding liquidity failed")
	})

	t.Run("ExecuteBtcToHbdSwap", func(t *testing.T) {
		initialBalance := getBalance(t, ctx, testEnv, testEnv.userAddr, "HBD")

		amountIn := int64(100000) // 0.001 BTC
		result, err := executeSwap(t, ctx, testEnv, "BTC", "HBD", amountIn)
		require.NoError(t, err, "Swap execution failed")
		require.Greater(t, result.AmountOut, int64(0), "Swap should return some HBD")

		finalBalance := getBalance(t, ctx, testEnv, testEnv.userAddr, "HBD")
		require.Greater(t, finalBalance, initialBalance, "HBD balance should increase after swap")
	})

	t.Run("ExecuteHbdSavingsToHbdSwap", func(t *testing.T) {
		initialBalance := getBalance(t, ctx, testEnv, testEnv.userAddr, "HBD")

		amountIn := int64(1000000) // 1 HBD_SAVINGS
		result, err := executeSwap(t, ctx, testEnv, "HBD_SAVINGS", "HBD", amountIn)
		require.NoError(t, err, "HBD Savings swap execution failed")
		require.Greater(t, result.AmountOut, int64(0), "Swap should return some HBD")

		finalBalance := getBalance(t, ctx, testEnv, testEnv.userAddr, "HBD")
		require.Greater(t, finalBalance, initialBalance, "HBD balance should increase after swap")
	})

	t.Run("ExecuteHiveToHbdSwap", func(t *testing.T) {
		initialBalance := getBalance(t, ctx, testEnv, testEnv.userAddr, "HBD")

		amountIn := int64(1000000) // 1 HIVE
		result, err := executeSwap(t, ctx, testEnv, "HIVE", "HBD", amountIn)
		require.NoError(t, err, "HIVE swap execution failed")
		require.Greater(t, result.AmountOut, int64(0), "Swap should return some HBD")

		finalBalance := getBalance(t, ctx, testEnv, testEnv.userAddr, "HBD")
		require.Greater(t, finalBalance, initialBalance, "HBD balance should increase after swap")
	})

	t.Run("ExecuteBtcToHiveSwap", func(t *testing.T) {
		initialBalance := getBalance(t, ctx, testEnv, testEnv.userAddr, "HIVE")

		amountIn := int64(100000) // 0.001 BTC
		result, err := executeSwap(t, ctx, testEnv, "BTC", "HIVE", amountIn)
		require.NoError(t, err, "BTC to HIVE swap execution failed")
		require.Greater(t, result.AmountOut, int64(0), "Swap should return some HIVE")
		require.Equal(t, 2, len(result.Route), "BTC->HIVE should be a 2-hop route")

		finalBalance := getBalance(t, ctx, testEnv, testEnv.userAddr, "HIVE")
		require.Greater(t, finalBalance, initialBalance, "HIVE balance should increase after swap")
	})

	t.Run("ExecuteHbdToBtcSwap", func(t *testing.T) {
		initialBalance := getBalance(t, ctx, testEnv, testEnv.userAddr, "BTC")

		amountIn := int64(1000000) // 1 HBD
		result, err := executeSwap(t, ctx, testEnv, "HBD", "BTC", amountIn)
		require.NoError(t, err, "Reverse swap execution failed")
		require.Greater(t, result.AmountOut, int64(0), "Swap should return some BTC")

		finalBalance := getBalance(t, ctx, testEnv, testEnv.userAddr, "BTC")
		require.Greater(t, finalBalance, initialBalance, "BTC balance should increase after swap")
	})

	t.Run("RequestWithdrawal", func(t *testing.T) {
		btcAddr := "bc1qtestaddress"
		amount := uint64(50000) // 0.0005 BTC

		err := requestWithdrawal(t, ctx, testEnv, amount, btcAddr)
		require.NoError(t, err, "Withdrawal request failed")

		// Verify BTC balance decreased
		finalBtcBalance := getBalance(t, ctx, testEnv, testEnv.userAddr, "BTC")
		require.Less(t, finalBtcBalance, testEnv.initialBtcBalance, "BTC balance should decrease after withdrawal request")
	})
}

func TestRouteComputation(t *testing.T) {
	// TODO: Test route finding algorithms
	t.Skip("Route computation tests not implemented")
}

func TestIndexerSync(t *testing.T) {
	// TODO: Test indexer event processing
	t.Skip("Indexer tests not implemented")
}

// Test environment
type TestEnvironment struct {
	vscEndpoint        string
	userAddr           string
	userKey            string
	initialBtcBalance  int64
	initialHbdBalance  int64
	poolID             string
	btcMappingContract string
	tokenRegistryContract string
	dexRouterContract  string
	cleanup            func()
}

func setupTestEnvironment(t *testing.T, ctx context.Context) *TestEnvironment {
	// Use environment variables or defaults for test configuration
	env := &TestEnvironment{
		vscEndpoint: getEnvOrDefault("VSC_TEST_ENDPOINT", "http://localhost:4000"),
		userAddr:    getEnvOrDefault("VSC_TEST_USER", "test-user"),
		userKey:     getEnvOrDefault("VSC_TEST_KEY", "test-key"),
	}

	// Record initial balances
	env.initialBtcBalance = getBalance(t, ctx, env, env.userAddr, "BTC")
	env.initialHbdBalance = getBalance(t, ctx, env, env.userAddr, "HBD")

	// Setup cleanup function
	env.cleanup = func() {
		// Cleanup test data/state
		t.Log("Cleaning up test environment")
	}

	return env
}

// Helper functions

func deployContracts(t *testing.T, ctx context.Context) error {
	// TODO: Use CLI to deploy contracts to test VSC node
	// For now, assume contracts are pre-deployed
	t.Log("Contracts assumed to be deployed")
	return nil
}

func registerBtcToken(t *testing.T, ctx context.Context, env *TestEnvironment) error {
	// TODO: Use SDK to register BTC token
	t.Log("BTC token assumed to be registered")
	return nil
}

func submitBtcHeaders(t *testing.T, ctx context.Context, env *TestEnvironment) error {
	// TODO: Start oracle service and submit test headers
	t.Log("BTC headers assumed to be submitted")
	return nil
}

func createTestDeposit(t *testing.T, ctx context.Context) (*DepositTx, error) {
	// TODO: Create actual BTC transaction for testing
	// For now, return mock data
	return &DepositTx{
		txid:   "mock_txid_12345",
		vout:   0,
		amount: 100000, // 0.001 BTC in satoshis
	}, nil
}

func generateSPVProof(t *testing.T, txid string, vout uint32) []byte {
	// TODO: Generate actual SPV proof
	// For now, return mock proof
	return []byte("mock_spv_proof_data")
}

func proveDeposit(t *testing.T, ctx context.Context, env *TestEnvironment, proof []byte) (uint64, error) {
	// TODO: Use SDK to submit deposit proof
	t.Logf("Submitting deposit proof: %x", proof)
	return 100000, nil // Mock minted amount
}

func createBtcHbdPool(t *testing.T, ctx context.Context, env *TestEnvironment) (string, error) {
	// TODO: Use SDK to create BTC/HBD pool
	t.Log("Creating BTC/HBD liquidity pool")
	return "btc-hbd-pool-123", nil
}

func addLiquidityToPool(t *testing.T, ctx context.Context, env *TestEnvironment, poolID string, btcAmount, hbdAmount int64) error {
	// TODO: Use SDK to add liquidity
	t.Logf("Adding liquidity to pool %s: %d BTC + %d HBD", poolID, btcAmount, hbdAmount)
	return nil
}

func executeSwap(t *testing.T, ctx context.Context, env *TestEnvironment, fromAsset, toAsset string, amountIn int64) (*SwapResult, error) {
	// TODO: Use SDK to compute and execute swap
	t.Logf("Executing swap: %d %s -> %s", amountIn, fromAsset, toAsset)

	// Mock route - single hop for direct routes, two hops for BTC->HIVE
	var route []PoolHop
	if fromAsset == "BTC" && toAsset == "HIVE" {
		route = []PoolHop{
			{PoolID: "btc-hbd-pool", TokenIn: "BTC", TokenOut: "HBD", AmountIn: amountIn, AmountOut: amountIn * 10},
			{PoolID: "hive-hbd-pool", TokenIn: "HBD", TokenOut: "HIVE", AmountIn: amountIn * 10, AmountOut: amountIn},
		}
	} else {
		route = []PoolHop{
			{PoolID: "mock-pool", TokenIn: fromAsset, TokenOut: toAsset, AmountIn: amountIn, AmountOut: amountIn * 10},
		}
	}

	return &SwapResult{
		Route:      route,
		AmountOut:  amountIn * 10, // Mock conversion rate
		PriceImpact: 0.01,
		Fee:         amountIn / 100,
	}, nil
}

func requestWithdrawal(t *testing.T, ctx context.Context, env *TestEnvironment, amount uint64, btcAddr string) error {
	// TODO: Use SDK to request withdrawal
	t.Logf("Requesting withdrawal: %d satoshis to %s", amount, btcAddr)
	return nil
}

func getBalance(t *testing.T, ctx context.Context, env *TestEnvironment, address, asset string) int64 {
	// TODO: Query balance from VSC node
	t.Logf("Getting balance for %s %s", address, asset)

	// Mock balances
	switch asset {
	case "BTC":
		return env.initialBtcBalance
	case "HBD":
		return env.initialHbdBalance
	default:
		return 0
	}
}

func waitForConfirmation(t *testing.T, txid string) {
	// TODO: Wait for transaction confirmation
	time.Sleep(2 * time.Second)
}

// Types
type DepositTx struct {
	txid   string
	vout   uint32
	amount uint64
}

type SwapResult struct {
	Route      []PoolHop
	AmountOut  int64
	PriceImpact float64
	Fee         int64
	TxPayload   []byte
}

type PoolHop struct {
	PoolID    string
	TokenIn   string
	TokenOut  string
	AmountIn  int64
	AmountOut int64
}

// Utility functions
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
