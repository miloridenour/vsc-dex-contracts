# DEX E2E Test Examples

This directory contains comprehensive examples and test scenarios for the VSC DEX Router contract.

## Files

- `dex-e2e-test.md` - Complete E2E test documentation with API calls and expected responses
- `test-dex-e2e.sh` - Automated test runner script
- `swap-btc-to-hbd.json` - Basic BTC to HBD swap example
- `swap-btc-to-hbd-savings.json` - BTC to HBD Savings swap
- `swap-with-referral.json` - Swap with referral fees
- `swap-with-return-address.json` - Swap with return address for refunds

## Running the Tests

### Prerequisites

1. **VSC Node**: Running locally on port 4000
2. **DEX Router Contract**: Deployed and accessible
3. **Router Service**: Running on port 8080
4. **Indexer Service**: Running on port 8081

### Environment Variables

```bash
export VSC_NODE=http://localhost:4000
export ROUTER_SERVICE=http://localhost:8080
export INDEXER_SERVICE=http://localhost:8081
```

### Run All Tests

```bash
cd /path/to/vsc-dex-mapping
./test-dex-e2e.sh
```

### Run Individual Tests

```bash
# Test pool creation
./test-dex-e2e.sh | grep "Create.*pool"

# Test swaps only
./test-dex-e2e.sh | grep "swap"
```

## Test Scenarios Covered

### 1. Pool Management
- ✅ Create HIVE-HBD pool
- ✅ Create BTC-HBD pool
- ✅ Add liquidity to pools
- ✅ Query pool information

### 2. Direct Swaps
- ✅ HBD → HIVE swap
- ✅ BTC → HBD swap
- ✅ Slippage protection
- ✅ Fee accumulation

### 3. Two-Hop Swaps
- ✅ BTC → HBD → HIVE routing
- ✅ Multi-pool calculations
- ✅ Intermediate amount handling

### 4. Error Handling
- ✅ Failed swaps with return addresses
- ✅ Invalid pool creation
- ✅ Slippage tolerance exceeded
- ✅ Insufficient reserves

### 5. Advanced Features
- ✅ Referral fees
- ✅ Fee claiming
- ✅ Liquidity withdrawal

## Manual Testing

### Using cURL

```bash
# Create a pool
curl -X POST http://localhost:8080/api/v1/contract/dex-router/create_pool \
  -H "Content-Type: application/json" \
  -d '{"asset0": "HBD", "asset1": "HIVE", "fee_bps": 8}'

# Add liquidity
curl -X POST http://localhost:8080/api/v1/contract/dex-router/execute \
  -H "Content-Type: application/json" \
  -d '{
    "type": "deposit",
    "version": "1.0.0",
    "asset_in": "HBD",
    "asset_out": "HIVE",
    "recipient": "alice",
    "metadata": {"amount0": "1000000", "amount1": "500000"}
  }'

# Execute swap
curl -X POST http://localhost:8080/api/v1/contract/dex-router/execute \
  -H "Content-Type: application/json" \
  -d '{
    "type": "swap",
    "version": "1.0.0",
    "asset_in": "BTC",
    "asset_out": "HIVE",
    "recipient": "bob",
    "min_amount_out": 95000,
    "slippage_bps": 150
  }'
```

### Using the Router Service

The router service provides a higher-level API for DEX operations:

```bash
# Get route for a swap
curl -X POST http://localhost:8080/api/v1/route \
  -H "Content-Type: application/json" \
  -d '{
    "fromAsset": "BTC",
    "toAsset": "HIVE",
    "amount": 10000
  }'
```

## Expected Results

### Pool Creation
- HIVE-HBD pool ID: 1
- BTC-HBD pool ID: 2
- Both pools have 8 bps (0.08%) fee

### Liquidity Addition
- HIVE-HBD pool: 1,000,000 HBD + 500,000 HIVE
- BTC-HBD pool: 2,000,000 HBD + 100,000 BTC
- LP tokens minted proportionally

### Swap Results
- Direct swaps: ~2-5% slippage depending on amount
- Two-hop swaps: ~4-10% total slippage
- Fees accumulated in respective pools

### Error Scenarios
- Invalid operations return error messages
- Failed swaps with return addresses trigger refund logic
- Insufficient reserves prevent swaps

## Monitoring

### Pool State
```bash
# Check all pools
curl http://localhost:8081/indexer/pools

# Check specific pool
curl -X POST http://localhost:8080/api/v1/contract/dex-router/get_pool \
  -H "Content-Type: application/json" \
  -d '"1"'
```

### Transaction Logs
```bash
# Monitor contract events
curl http://localhost:4000/api/v1/graphql \
  -H "Content-Type: application/json" \
  -d '{
    "query": "subscription {
      contractEvents(contractId: \"dex-router\") {
        method args timestamp
      }
    }"
  }'
```

## Troubleshooting

### Common Issues

1. **Services not available**: Check that VSC node, router, and indexer are running
2. **Pool creation fails**: Ensure assets are different and valid
3. **Swaps fail**: Check pool reserves and slippage settings
4. **Liquidity operations fail**: Verify user has sufficient balance

### Debug Commands

```bash
# Check service health
curl http://localhost:8080/health
curl http://localhost:8081/health

# Check VSC node
curl http://localhost:4000/api/v1/graphql \
  -H "Content-Type: application/json" \
  -d '{"query": "{ block { height } }"}'
```

## Performance Benchmarks

- Pool creation: < 100ms
- Direct swap: < 50ms
- Two-hop swap: < 150ms
- Liquidity operations: < 200ms
- Query operations: < 20ms

## Extending the Tests

To add new test scenarios:

1. Add test case to `dex-e2e-test.md`
2. Implement test logic in `test-dex-e2e.sh`
3. Add API call helpers as needed
4. Update verification logic

The test framework is designed to be easily extensible for new DEX features.
