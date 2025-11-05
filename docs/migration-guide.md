# Migration Guide: go-vsc-node DEX → External Services

This guide explains how to migrate DEX functionality from `go-vsc-node` internals to the external `vsc-dex-mapping` services.

## Current Architecture (go-vsc-node)

The current DEX implementation includes:
- `modules/dex-router/router.go`: Core routing logic with bonding curves
- `modules/dex-router/token_registry.go`: Native asset registry
- `modules/dex-router/bonding_curve.go`: Pool bonding/locking logic
- Database schemas in `modules/db/vsc/dex/`

## Migration Strategy

### Phase 1: Extract Router Logic

**From:** `go-vsc-node/modules/dex-router/router.go`
**To:** `vsc-dex-mapping/services/router/`

#### Files to migrate:

1. **Route computation logic** (`ExecuteSwap`, `ExecuteCrossChainSwap`)
   - Move to `services/router/router.go`
   - Replace database dependencies with indexer HTTP calls
   - Update to use external contract ABIs

2. **Pool management** (`RegisterPool`, `GetPoolInfo`)
   - Move to `services/router/pool_manager.go`
   - Adapt for external pool discovery via indexer

3. **Cross-chain operations** (`GeneratePairAccount`, bridge actions)
   - Move to `services/router/cross_chain.go`
   - Integrate with mapping contract workflows

### Phase 2: Extract Token Registry

**From:** `go-vsc-node/modules/dex-router/token_registry.go`
**To:** `vsc-dex-mapping/contracts/token-registry/`

#### Migration steps:

1. Convert Go token registry to TinyGo contract
2. Move native asset initialization to contract deployment
3. Update router to query indexer for token metadata

### Phase 3: Extract Bonding Curves

**From:** `go-vsc-node/modules/dex-router/bonding_curve.go`
**To:** `vsc-dex-mapping/services/router/bonding.go`

#### Files to migrate:

1. **Bonding curve calculations** → Router service
2. **Withdrawal limits** → Router service
3. **Pool status management** → Indexer read models

### Phase 4: Database Migration

**Current:** Internal database schemas
**New:** Indexer projections + contract state

#### Migration steps:

1. Replace internal DB queries with indexer HTTP APIs
2. Move pool/token data to indexer read models
3. Use contract state for authoritative data

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
