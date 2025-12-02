# DEX Router Contract

A unified smart contract that implements a complete decentralized exchange with automated market making (AMM) functionality. This contract owns and manages all liquidity pools internally, providing swap, deposit, and withdrawal operations through a standardized JSON interface.

## Features

- **Unified Pool Management**: Single contract manages all liquidity pools
- **Constant Product AMM**: x*y=k formula with configurable fees
- **JSON Schema Interface**: Standardized payload format for all operations
- **Multi-Hop Routing**: Support for complex swap routes
- **Slippage Protection**: Configurable minimum output amounts
- **Referral System**: Optional referral fees for swaps
- **Fee Collection**: Accumulated fees claimable by system

## Operations

### Initialize Contract
```json
{
  "action": "init",
  "payload": "1.0.0"
}
```

### Create Pool
```json
{
  "action": "create_pool",
  "payload": "{\"asset0\": \"HBD\", \"asset1\": \"HIVE\", \"fee_bps\": 8}"
}
```

### Execute Swap
```json
{
  "action": "execute",
  "payload": {
    "type": "swap",
    "version": "1.0.0",
    "asset_in": "HBD",
    "asset_out": "HIVE",
    "recipient": "hive:user123",
    "min_amount_out": 1000000,
    "slippage_bps": 50,
    "beneficiary": "hive:referrer",
    "ref_bps": 25
  }
}
```

### Add Liquidity (Deposit)
```json
{
  "action": "execute",
  "payload": {
    "type": "deposit",
    "version": "1.0.0",
    "asset_in": "HBD",
    "asset_out": "HIVE",
    "recipient": "hive:user123"
  }
}
```

### Remove Liquidity (Withdrawal)
```json
{
  "action": "execute",
  "payload": {
    "type": "withdrawal",
    "version": "1.0.0",
    "asset_in": "HBD",
    "asset_out": "HIVE",
    "recipient": "hive:user123"
  }
}
```

### Query Pool
```json
{
  "action": "get_pool",
  "payload": "1"
}
```

### Claim Fees (System Only)
```json
{
  "action": "claim_fees",
  "payload": "1"
}
```

## Building

```bash
cd contracts/dex-router
tinygo build -o ../../bin/dex-router.wasm -target wasm main.go utils.go
```

## Architecture

The DEX Router contract maintains all pool state internally using namespaced keys:

- `pool/{poolId}/asset0` - First asset in pair
- `pool/{poolId}/asset1` - Second asset in pair
- `pool/{poolId}/reserve0` - Reserve amount of asset0
- `pool/{poolId}/reserve1` - Reserve amount of asset1
- `pool/{poolId}/fee` - Fee in basis points
- `pool/{poolId}/total_lp` - Total LP tokens minted
- `pool/{poolId}/lp/{address}` - LP balance for address
- `pool/{poolId}/fee0` - Accumulated fees for asset0
- `pool/{poolId}/fee1` - Accumulated fees for asset1

## Security

- **Slippage Protection**: Enforced minimum output validation
- **Reserve Validation**: Prevents swaps exceeding pool reserves
- **Fee Bounds**: Configurable fee limits (0-100%)
- **System Operations**: Fee claiming restricted to system accounts
- **Asset Validation**: Ensures valid asset pairs and amounts
