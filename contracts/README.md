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
