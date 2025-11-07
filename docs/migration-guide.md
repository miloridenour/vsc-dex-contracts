# Migration Guide: go-vsc-node DEX → External Services ✅

This guide explains how DEX functionality has been migrated from `go-vsc-node` internals to the external `vsc-dex-mapping` services.

## ✅ Migration Status: Complete

The core DEX routing logic, pool management, and bonding curve functionality have been successfully migrated from `go-vsc-node` to the external services architecture.

## What Was Migrated

### ✅ Core DEX Router Logic
**From:** `go-vsc-node/modules/dex-router/router.go`
**To:** `vsc-dex-mapping/services/router/router.go`

- **ExecuteSwap**: Two-hop swap orchestration through HBD
- **executeDirectSwap**: Direct pool swap execution
- **findPool**: Pool discovery logic
- **calculateExpectedOutput**: AMM output calculations
- **calculateMinOutput**: Slippage protection
- **GeneratePairAccount**: Cross-chain account generation

### ✅ Pool Management
- Pool information structures and validation
- Reserve tracking and AMM calculations
- Cross-chain bridge action handling

### ✅ Token Support
- HBD, HIVE, HBD_SAVINGS native asset support
- BTC mapping integration
- Multi-chain asset handling

### ✅ Route Computation
- Direct routes (1-hop)
- HBD-mediated routes (2-hop)
- Unsupported route detection

## Architecture Changes

### Before (Internal)
```go
// Direct database access
pools, err := r.dexDb.GetPoolsByAsset(asset0)
bondingMgr := r.bondingCurveMgr
```

### After (External)
```go
// External API calls
pools, err := indexerClient.GetPools(asset0)
bondingMgr := router.NewBondingManager(indexerClient)
```

### Database Migration
- **Internal DB schemas** → **Indexer read models**
- **Direct queries** → **HTTP API calls**
- **Synchronous operations** → **Eventual consistency**

## Implementation Details

### Router Service Migration

```go
// Before (internal)
func (r *Router) ExecuteSwap(params dex.SwapParams) (*dex.SwapResult, error) {
    pool, err := r.dexDb.GetPoolInfo(contractId)
    // ... direct DB access
}

// After (external)
func (r *Router) ExecuteSwap(params SwapParams) (*SwapResult, error) {
    pool, err := r.indexerClient.GetPool(contractId)
    // ... HTTP API calls
}
```

### Token Registry Migration

```go
// Before (internal)
func (tr *TokenRegistry) GetTokenInfo(asset string) (*dex.TokenMetadata, error) {
    return tr.dexDb.GetTokenMetadata(asset)
}

// After (external contract)
func getToken(symbol []byte) []byte {
    // Contract storage access
    state := sdk.GetState("tokens")
    // Return JSON-encoded token info
}
```

### Bonding Curve Migration

```go
// Before (internal)
func (bcm *BondingCurveManager) UpdateBondingMetric(contractId string, newReserve0, newReserve1 uint64) error {
    return bcm.dexDb.UpdateBondingMetric(contractId, metric)
}

// After (external)
func (r *Router) UpdateBondingMetric(poolID string, reserve0, reserve1 uint64) error {
    // Update via indexer API
    return r.indexerClient.UpdatePoolMetrics(poolID, reserve0, reserve1)
}
```

## Breaking Changes

### API Changes

1. **Direct DB access** → **HTTP APIs**
   - `dexDb.GetPoolInfo()` → `indexerClient.GetPool()`
   - `dexDb.GetTokenMetadata()` → `indexerClient.GetToken()`

2. **Internal events** → **External subscriptions**
   - Database triggers → GraphQL subscriptions
   - Internal event handlers → Indexer event processors

3. **Synchronous operations** → **Asynchronous workflows**
   - Immediate DB updates → Eventual consistency via indexer

### Configuration Changes

1. **Database connections** → **Indexer endpoints**
2. **Internal module dependencies** → **External service dependencies**
3. **Single binary** → **Multi-service deployment**

## Testing Migration

### Unit Tests
- Extract existing unit tests to new service packages
- Update mock dependencies for external APIs
- Add integration tests for service communication

### E2E Tests
- Extend `e2e/e2e_test.go` with full DEX workflows
- Test cross-service communication
- Validate data consistency between services

## Deployment Considerations

### Service Dependencies
```
Router Service → Indexer API
Router Service → VSC GraphQL
Oracle Service → Bitcoin RPC
Indexer Service → VSC GraphQL Subscriptions
```

### Environment Variables
```bash
# VSC connection
VSC_ENDPOINT=http://localhost:4000
VSC_GRAPHQL_WS=ws://localhost:4000/graphql

# Bitcoin oracle
BTC_RPC_HOST=localhost:8332
BTC_RPC_USER=user
BTC_RPC_PASS=pass

# Service ports
ROUTER_PORT=8080
INDEXER_PORT=8081
ORACLE_PORT=8082
```

## Rollback Plan

If migration issues arise:

1. **Keep old implementation** alongside new services
2. **Gradual rollout** with feature flags
3. **Dual writes** during transition period
4. **Monitoring** for data consistency

## Future Extensions

Once migrated, the external architecture enables:

- **Multi-chain support**: Easy addition of SOL/ETH oracles
- **DEX aggregators**: Route across multiple DEXes
- **Third-party integrations**: Open APIs for external developers
- **Scalability**: Independent service scaling
- **Upgrades**: Zero-downtime service updates
