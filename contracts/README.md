# Contracts

TinyGo smart contracts deployed on VSC for cross-chain asset mapping and DEX operations.

## btc-mapping

Bitcoin UTXO mapping contract that:
- Accepts Bitcoin block headers for SPV verification
- Processes deposit proofs to mint mapped BTC tokens
- Handles withdrawal requests to burn tokens and authorize BTC spends

### Building
```bash
cd btc-mapping
tinygo build -o ../../bin/btc-mapping.wasm -target wasm main.go
```

## v2-amm

HBD-anchored AMM (Automated Market Maker) contract with slip-adjusted fees:
- **Constant product formula** (x*y=k) for swap calculations
- **HBD-anchored pools**: Every pool is anchored to HBD (asset0)
- **Base fees**: Applied only when input side is HBD (default 8 bps = 0.08%)
- **Slip-adjusted fees**: Optional portion of slippage above baseline kept for LPs
- **Liquidity provision**: Add/remove liquidity with LP token minting/burning
- **Referral support**: Optional referral fees (0.01%-10.00%) for swaps
- **Fee claiming**: System-only function to claim accumulated HBD fees

### Features
- **Initialization**: `init` with payload `asset0,asset1,baseFeeBps` (e.g., `hbd,hive,8`)
- **Liquidity**: `add_liquidity amt0,amt1`, `remove_liquidity lpAmount`, `donate amt0,amt1`
- **Swaps**: `swap dir,amountIn[,minOut]` where `dir` is `0to1` or `1to0`
- **LP Management**: `transfer toAddress,amount`, `burn lpAmount`
- **System Functions**: `claim_fees`, `set_base_fee`, `set_slip_params`, `si_withdraw`

### Building
```bash
cd v2-amm
tinygo build -o ../../bin/v2-amm.wasm -target wasm main.go
```

**Note**: This contract is integrated from the `go-contract-template` examples and provides the actual DEX swap functionality that the router service calls.

## token-registry

Registry contract for wrapped/mapped assets:
- Registers asset metadata (symbol, decimals, owner)
- Enforces ownership restrictions for mapped tokens
- Provides token discovery for DEX operations

### Building
```bash
cd token-registry
tinygo build -o ../../bin/token-registry.wasm -target wasm main.go
```
