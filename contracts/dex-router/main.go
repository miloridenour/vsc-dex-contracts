package main

import (
	sdk "dex-router/sdk"
	"encoding/json"
	"math/bits"
	"strconv"
)

func main() {}

// DEX Instruction Schema
type DexInstruction struct {
	Type          string                 `json:"type"`
	Version       string                 `json:"version"`
	AssetIn       string                 `json:"asset_in"`
	AssetOut      string                 `json:"asset_out"`
	Recipient     string                 `json:"recipient"`
	SlippageBps   *int                   `json:"slippage_bps,omitempty"`
	MinAmountOut  *int64                 `json:"min_amount_out,omitempty"`
	Beneficiary   *string                `json:"beneficiary,omitempty"`
	RefBps        *int                   `json:"ref_bps,omitempty"`
	ReturnAddress *ReturnAddress         `json:"return_address,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

type ReturnAddress struct {
	Chain   string `json:"chain"`
	Address string `json:"address"`
}

// Contract initialization
// Payload: version string (e.g. "1.0.0")
//
//go:wasmexport init
func Init(payload *string) *string {
	if payload == nil || *payload == "" {
		setStr(keyVersion, "1.0.0")
	} else {
		setStr(keyVersion, *payload)
	}
	setUint(keyNextPoolId, 1)
	return nil
}

// Create a new liquidity pool
// Payload: JSON with pool parameters
// {"asset0": "HBD", "asset1": "HIVE", "fee_bps": 8}
//

//go:wasmexport create_pool
func CreatePool(payload *string) *string {
	if payload == nil {
		return &[]string{"error", "payload required"}[1]
	}

	var params struct {
		Asset0 string `json:"asset0"`
		Asset1 string `json:"asset1"`
		FeeBps uint64 `json:"fee_bps"`
	}

	if err := json.Unmarshal([]byte(*payload), &params); err != nil {
		return &[]string{"error", "invalid payload"}[1]
	}

	// Validate assets are different
	if params.Asset0 == params.Asset1 {
		return &[]string{"error", "assets must be different"}[1]
	}

	// Default fee if not specified
	if params.FeeBps == 0 {
		params.FeeBps = defaultBaseFeeBps
	}

	// Generate pool ID
	poolId := strconv.FormatUint(getUint(keyNextPoolId), 10)
	setUint(keyNextPoolId, getUint(keyNextPoolId)+1)

	// Initialize pool state
	setPoolAsset0(poolId, params.Asset0)
	setPoolAsset1(poolId, params.Asset1)
	setPoolReserve0(poolId, 0)
	setPoolReserve1(poolId, 0)
	setPoolFee(poolId, params.FeeBps)
	setPoolTotalLp(poolId, 0)
	setUint(poolFee0Key(poolId), 0)
	setUint(poolFee1Key(poolId), 0)
	setStr(poolFeeLastClaimKey(poolId), sdk.GetEnv().Timestamp)

	return nil
}

// Execute DEX operation based on JSON schema
// Payload: JSON instruction as defined in schema
//
//go:wasmexport execute
func Execute(payload *string) *string {
	if payload == nil {
		return &[]string{"error", "payload required"}[1]
	}

	var instruction DexInstruction
	if err := json.Unmarshal([]byte(*payload), &instruction); err != nil {
		return &[]string{"error", "invalid json payload"}[1]
	}

	// Validate required fields
	if instruction.Type == "" || instruction.Version == "" ||
		instruction.AssetIn == "" || instruction.AssetOut == "" ||
		instruction.Recipient == "" {
		return &[]string{"error", "missing required fields"}[1]
	}

	switch instruction.Type {
	case "swap":
		return executeSwap(instruction)
	case "deposit":
		return executeDeposit(instruction)
	case "withdrawal":
		return executeWithdrawal(instruction)
	default:
		return &[]string{"error", "unknown instruction type"}[1]
	}
}

// Execute swap operation
func executeSwap(instruction DexInstruction) *string {
	// Find direct pool first
	directPoolId := findPool(instruction.AssetIn, instruction.AssetOut)
	if directPoolId != "" {
		return executeDirectSwap(directPoolId, instruction)
	}

	// Try two-hop swap via HBD
	if instruction.AssetIn != "HBD" && instruction.AssetOut != "HBD" {
		return executeTwoHopSwap(instruction)
	}

	return &[]string{"error", "no suitable pool found"}[1]
}

// Find pool by assets - iterates through all pools to find matching pair
func findPool(assetA, assetB string) string {
	nextPoolId := getUint(keyNextPoolId)
	for i := uint64(1); i < nextPoolId; i++ {
		poolId := strconv.FormatUint(i, 10)
		asset0 := getPoolAsset0(poolId)
		asset1 := getPoolAsset1(poolId)
		if asset0 == "" {
			continue // Pool doesn't exist
		}
		// Check both asset orders
		if (asset0 == assetA && asset1 == assetB) || (asset0 == assetB && asset1 == assetA) {
			return poolId
		}
	}
	return ""
}

// Execute direct swap within a pool
func executeDirectSwap(poolId string, instruction DexInstruction) *string {
	asset0 := getPoolAsset0(poolId)
	asset1 := getPoolAsset1(poolId)

	if asset0 == "" {
		return &[]string{"error", "pool not found"}[1]
	}

	r0 := getPoolReserve0(poolId)
	r1 := getPoolReserve1(poolId)
	feeBps := getPoolFee(poolId)

	if r0 == 0 || r1 == 0 {
		return &[]string{"error", "pool has zero reserves"}[1]
	}

	// For now, use MinAmountOut as the input amount (this is a simplification)
	var amountInU uint64
	if instruction.MinAmountOut != nil {
		amountInU = uint64(*instruction.MinAmountOut)
	} else {
		return &[]string{"error", "amount_in required for swap"}[1]
	}

	var amountOut uint64
	var inputAsset, outputAsset string
	var feeReserveKey string

	// Determine swap direction and calculate output
	if asset0 == instruction.AssetIn && asset1 == instruction.AssetOut {
		// asset0 -> asset1
		inputAsset = asset0
		outputAsset = asset1
		feeReserveKey = poolFee0Key(poolId)

		// Calculate output: dy = r1 - (r0 * r1) / (r0 + dx)
		dx := amountInU * (10000 - feeBps) / 10000 // Apply fee
		if dx == 0 {
			dx = 1
		}
		k := r0 * r1
		newR0 := r0 + dx
		amountOut = r1 - (k / newR0)

		// Update reserves
		setPoolReserve0(poolId, newR0)
		setPoolReserve1(poolId, r1-amountOut)

	} else if asset1 == instruction.AssetIn && asset0 == instruction.AssetOut {
		// asset1 -> asset0
		inputAsset = asset1
		outputAsset = asset0
		feeReserveKey = poolFee1Key(poolId)

		// Calculate output: dx = r0 - (r0 * r1) / (r1 + dy)
		dy := amountInU // No fee for non-HBD input
		k := r0 * r1
		newR1 := r1 + dy
		amountOut = r0 - (k / newR1)

		// Update reserves
		setPoolReserve1(poolId, newR1)
		setPoolReserve0(poolId, r0-amountOut)

	} else {
		return &[]string{"error", "invalid asset pair for pool"}[1]
	}

	// Apply slippage protection if specified
	if instruction.SlippageBps != nil {
		minOut := amountOut * (10000 - uint64(*instruction.SlippageBps)) / 10000
		if amountOut < minOut {
			return &[]string{"error", "slippage tolerance exceeded"}[1]
		}
	}

	// Draw input asset and transfer output asset
	drawAsset(int64(amountInU), inputAsset)

	// Handle referral fees
	if instruction.Beneficiary != nil && instruction.RefBps != nil {
		refOut := amountOut * uint64(*instruction.RefBps) / 10000
		if refOut > 0 {
			if refOut >= amountOut {
				refOut = amountOut - 1
			}
			amountOut -= refOut
			transferAsset(*instruction.Beneficiary, int64(refOut), outputAsset)
		}
	}

	transferAsset(instruction.Recipient, int64(amountOut), outputAsset)

	// Accumulate fees (simplified - only for HBD input)
	if inputAsset == "HBD" {
		fee := amountInU - (amountInU * (10000 - feeBps) / 10000)
		if fee > 0 {
			currentFee := getUint(feeReserveKey)
			setUint(feeReserveKey, currentFee+fee)
		}
	}

	return nil
}

// Execute two-hop swap via HBD
func executeTwoHopSwap(instruction DexInstruction) *string {
	// Find first pool: AssetIn -> HBD
	pool1Id := findPool(instruction.AssetIn, "HBD")
	if pool1Id == "" {
		return &[]string{"error", "no pool found for first hop"}[1]
	}

	// Find second pool: HBD -> AssetOut
	pool2Id := findPool("HBD", instruction.AssetOut)
	if pool2Id == "" {
		return &[]string{"error", "no pool found for second hop"}[1]
	}

	// Get pool information
	asset1_0 := getPoolAsset0(pool1Id)
	r1_0 := getPoolReserve0(pool1Id)
	r1_1 := getPoolReserve1(pool1Id)
	fee1 := getPoolFee(pool1Id)

	r2_0 := getPoolReserve0(pool2Id)
	r2_1 := getPoolReserve1(pool2Id)
	fee2 := getPoolFee(pool2Id)

	// Determine input amount (simplified - using MinAmountOut as input for now)
	// In a real implementation, this would be passed explicitly or calculated
	var amountIn uint64
	if instruction.MinAmountOut != nil {
		amountIn = uint64(*instruction.MinAmountOut)
	} else {
		return &[]string{"error", "amount_in required for two-hop swap"}[1]
	}

	// Calculate first hop: AssetIn -> HBD
	var amountIntermediate uint64
	if asset1_0 == instruction.AssetIn {
		// AssetIn is asset0, HBD is asset1
		k1 := r1_0 * r1_1
		dxEff := amountIn * (10000 - fee1) / 10000
		if dxEff == 0 {
			dxEff = 1
		}
		newR0 := r1_0 + dxEff
		amountIntermediate = r1_1 - (k1 / newR0)

		// Update first pool reserves
		setPoolReserve0(pool1Id, newR0)
		setPoolReserve1(pool1Id, r1_1-amountIntermediate)
	} else {
		// AssetIn is asset1, HBD is asset0
		k1 := r1_0 * r1_1
		dyEff := amountIn * (10000 - fee1) / 10000
		if dyEff == 0 {
			dyEff = 1
		}
		newR1 := r1_1 + dyEff
		amountIntermediate = r1_0 - (k1 / newR1)

		// Update first pool reserves
		setPoolReserve1(pool1Id, newR1)
		setPoolReserve0(pool1Id, r1_0-amountIntermediate)
	}

	// Calculate second hop: HBD -> AssetOut
	var amountOut uint64
	if getPoolAsset0(pool2Id) == "HBD" {
		// HBD is asset0, AssetOut is asset1
		k2 := r2_0 * r2_1
		dxEff := amountIntermediate * (10000 - fee2) / 10000
		if dxEff == 0 {
			dxEff = 1
		}
		newR0 := r2_0 + dxEff
		amountOut = r2_1 - (k2 / newR0)

		// Update second pool reserves
		setPoolReserve0(pool2Id, newR0)
		setPoolReserve1(pool2Id, r2_1-amountOut)
	} else {
		// HBD is asset1, AssetOut is asset0
		k2 := r2_0 * r2_1
		dyEff := amountIntermediate * (10000 - fee2) / 10000
		if dyEff == 0 {
			dyEff = 1
		}
		newR1 := r2_1 + dyEff
		amountOut = r2_0 - (k2 / newR1)

		// Update second pool reserves
		setPoolReserve1(pool2Id, newR1)
		setPoolReserve0(pool2Id, r2_0-amountOut)
	}

	// Apply slippage protection
	if instruction.SlippageBps != nil {
		minOut := amountOut * (10000 - uint64(*instruction.SlippageBps)) / 10000
		if amountOut < minOut {
			return &[]string{"error", "slippage tolerance exceeded"}[1]
		}
	}

	// Execute the transfers
	drawAsset(int64(amountIn), instruction.AssetIn)
	transferAsset(instruction.Recipient, int64(amountOut), instruction.AssetOut)

	// Accumulate fees (simplified - only for HBD in first hop)
	if instruction.AssetIn == "HBD" {
		fee := amountIn - (amountIn * (10000 - fee1) / 10000)
		if fee > 0 {
			if asset1_0 == "HBD" {
				setUint(poolFee0Key(pool1Id), getUint(poolFee0Key(pool1Id))+fee)
			} else {
				setUint(poolFee1Key(pool1Id), getUint(poolFee1Key(pool1Id))+fee)
			}
		}
	}

	return nil
}

// Execute deposit (add liquidity)
func executeDeposit(instruction DexInstruction) *string {
	// Find the pool
	poolId := findPool(instruction.AssetIn, instruction.AssetOut)
	if poolId == "" {
		return &[]string{"error", "pool not found"}[1]
	}

	// For now, require deposit amounts to be specified in metadata
	// In a real implementation, this might come from transaction intents
	if instruction.Metadata == nil {
		return &[]string{"error", "deposit amounts required in metadata"}[1]
	}

	amt0Interface, ok := instruction.Metadata["amount0"]
	if !ok {
		return &[]string{"error", "amount0 required in metadata"}[1]
	}
	amt1Interface, ok := instruction.Metadata["amount1"]
	if !ok {
		return &[]string{"error", "amount1 required in metadata"}[1]
	}

	amt0Float, ok := amt0Interface.(float64)
	if !ok {
		return &[]string{"error", "amount0 must be number"}[1]
	}
	amt1Float, ok := amt1Interface.(float64)
	if !ok {
		return &[]string{"error", "amount1 must be number"}[1]
	}

	amt0U := uint64(amt0Float)
	amt1U := uint64(amt1Float)

	return executeAddLiquidity(poolId, amt0U, amt1U, instruction.Recipient)
}

// Execute withdrawal (remove liquidity)
func executeWithdrawal(instruction DexInstruction) *string {
	// Find the pool
	poolId := findPool(instruction.AssetIn, instruction.AssetOut)
	if poolId == "" {
		return &[]string{"error", "pool not found"}[1]
	}

	// For now, require LP amount to be specified in metadata
	if instruction.Metadata == nil {
		return &[]string{"error", "lp_amount required in metadata"}[1]
	}

	lpAmountInterface, ok := instruction.Metadata["lp_amount"]
	if !ok {
		return &[]string{"error", "lp_amount required in metadata"}[1]
	}

	lpAmountFloat, ok := lpAmountInterface.(float64)
	if !ok {
		return &[]string{"error", "lp_amount must be number"}[1]
	}

	lpAmountU := uint64(lpAmountFloat)

	return executeRemoveLiquidity(poolId, lpAmountU, instruction.Recipient)
}

// Execute add liquidity operation
func executeAddLiquidity(poolId string, amt0U, amt1U uint64, provider string) *string {
	asset0 := getPoolAsset0(poolId)
	asset1 := getPoolAsset1(poolId)

	// Pull funds from user intents into contract
	if amt0U > 0 {
		drawAsset(int64(amt0U), asset0)
	}
	if amt1U > 0 {
		drawAsset(int64(amt1U), asset1)
	}

	// Update reserves and mint LP
	r0 := getPoolReserve0(poolId)
	r1 := getPoolReserve1(poolId)
	totalLP := getPoolTotalLp(poolId)

	var minted uint64
	if totalLP == 0 {
		// Geometric mean using 128-bit product for first liquidity
		hi, lo := bits.Mul64(amt0U, amt1U)
		minted = sqrt128(hi, lo)
	} else {
		// Proportional minting
		m0 := amt0U * totalLP / r0
		m1 := amt1U * totalLP / r1
		minted = min64(m0, m1)
	}
	assertCustom(minted > 0)

	// Update state
	setPoolReserve0(poolId, r0+amt0U)
	setPoolReserve1(poolId, r1+amt1U)
	setPoolTotalLp(poolId, totalLP+minted)

	// Mint LP tokens to provider
	currentLP := getPoolLp(poolId, provider)
	setPoolLp(poolId, provider, currentLP+minted)

	return nil
}

// Execute remove liquidity operation
func executeRemoveLiquidity(poolId string, lpAmountU uint64, provider string) *string {
	providerAddr := sdk.Address(provider)
	userLP := getPoolLp(poolId, providerAddr.String())
	totalLP := getPoolTotalLp(poolId)

	assertCustom(lpAmountU > 0 && lpAmountU <= userLP && totalLP > 0)

	r0 := getPoolReserve0(poolId)
	r1 := getPoolReserve1(poolId)

	// Calculate proportional amounts
	amt0 := int64(r0 * lpAmountU / totalLP)
	amt1 := int64(r1 * lpAmountU / totalLP)

	// Update state first
	setPoolLp(poolId, providerAddr.String(), userLP-lpAmountU)
	setPoolTotalLp(poolId, totalLP-lpAmountU)
	setPoolReserve0(poolId, r0-uint64(amt0))
	setPoolReserve1(poolId, r1-uint64(amt1))

	// Transfer assets out
	asset0 := getPoolAsset0(poolId)
	asset1 := getPoolAsset1(poolId)
	if amt0 > 0 {
		transferAsset(provider, amt0, asset0)
	}
	if amt1 > 0 {
		transferAsset(provider, amt1, asset1)
	}

	return nil
}

// Helper functions

func calculateSwapOutput(amountIn, reserveIn, reserveOut, feeBps uint64, isAsset0Input bool) uint64 {
	feeMultiplier := uint64(10000 - feeBps)
	amountInAfterFee := amountIn * feeMultiplier / 10000
	if amountInAfterFee == 0 {
		amountInAfterFee = 1
	}

	k := reserveIn * reserveOut
	newReserveIn := reserveIn + amountInAfterFee
	amountOut := reserveOut - (k / newReserveIn)

	return amountOut
}

func applySlippageFee(amountOut, amountIn, amountInAfterFee, reserveIn, reserveOut uint64, isAsset0Input bool) uint64 {
	// Simplified slippage fee calculation
	// In full implementation, would calculate based on price impact
	return amountOut
}

// Query pool information
// Payload: pool_id
//
//go:wasmexport get_pool
func GetPool(payload *string) *string {
	if payload == nil {
		return &[]string{"error", "pool_id required"}[1]
	}

	poolId := *payload
	asset0 := getPoolAsset0(poolId)
	if asset0 == "" {
		return &[]string{"error", "pool not found"}[1]
	}

	poolInfo := map[string]interface{}{
		"asset0":   asset0,
		"asset1":   getPoolAsset1(poolId),
		"reserve0": getPoolReserve0(poolId),
		"reserve1": getPoolReserve1(poolId),
		"fee":      getPoolFee(poolId),
		"total_lp": getPoolTotalLp(poolId),
	}

	jsonBytes, err := json.Marshal(poolInfo)
	if err != nil {
		return &[]string{"error", "serialization failed"}[1]
	}

	result := string(jsonBytes)
	return &result
}

// Claim fees (system only)
// Payload: pool_id
//
//go:wasmexport claim_fees
func ClaimFees(payload *string) *string {
	if !isSystemSender() {
		return &[]string{"error", "system only"}[1]
	}

	if payload == nil {
		return &[]string{"error", "pool_id required"}[1]
	}

	poolId := *payload
	asset0 := getPoolAsset0(poolId)
	asset1 := getPoolAsset1(poolId)
	dao := sdk.Address("system:fr_balance")

	f0 := getUint(poolFee0Key(poolId))
	f1 := getUint(poolFee1Key(poolId))

	if f0 > 0 && isHbd(asset0) {
		setUint(poolFee0Key(poolId), 0)
		sdk.HiveWithdraw(dao, int64(f0), sdk.Asset(asset0))
	}
	if f1 > 0 && isHbd(asset1) {
		setUint(poolFee1Key(poolId), 0)
		sdk.HiveWithdraw(dao, int64(f1), sdk.Asset(asset1))
	}

	setStr(poolFeeLastClaimKey(poolId), sdk.GetEnv().Timestamp)
	return nil
}
