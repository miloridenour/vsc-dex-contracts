# Services

External services that integrate with VSC via public APIs (GraphQL, broadcast endpoints).

## oracle

Bitcoin oracle service that:
- Connects to Bitcoin node (btcd/bitcoind)
- Submits block headers to btc-mapping contract
- Verifies and forwards SPV deposit proofs

### Running
```bash
cd oracle
go run cmd/main.go --btc-host localhost:8332 --vsc-node http://localhost:4000
```

## router

DEX router adapter that:
- Computes optimal swap routes across VSC pools
- Composes transaction payloads for DEX operations
- Handles cross-chain swap orchestration

### Running
```bash
cd router
go run cmd/main.go --vsc-node http://localhost:4000
```

## indexer

Read model indexer that:
- Subscribes to VSC GraphQL events
- Builds projections for pools, tokens, and bridge operations
- Exposes REST/GraphQL APIs for frontend consumption

### Running
```bash
cd indexer
go run cmd/main.go --vsc-graphql ws://localhost:4000/graphql
```
