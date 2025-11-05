# Getting Started

## Prerequisites

- Go 1.21+
- TinyGo (for contract compilation)
- Bitcoin Core or btcd (for oracle)
- VSC node running locally or remote endpoint

## Setup

1. **Clone and setup**:
   ```bash
   git clone <repo-url>
   cd vsc-dex-mapping
   make setup
   ```

2. **Build contracts**:
   ```bash
   make contracts
   ```

3. **Deploy contracts** (requires VSC node access):
   ```bash
   go run cli/main.go deploy --vsc-endpoint http://localhost:4000 --key <your-key>
   ```

4. **Start services**:
   ```bash
   # Terminal 1: Oracle
   go run services/oracle/cmd/main.go --btc-host localhost:8332 --vsc-node http://localhost:4000

   # Terminal 2: Router
   go run services/router/cmd/main.go --vsc-node http://localhost:4000

   # Terminal 3: Indexer
   go run services/indexer/cmd/main.go --vsc-ws ws://localhost:4000/graphql
   ```

## Testing

Run the full E2E test suite:
```bash
make e2e
```

Run unit tests:
```bash
make test
```

## Development Workflow

1. **Contract changes**:
   ```bash
   cd contracts/btc-mapping
   # Edit main.go
   make contracts
   go run cli/main.go deploy
   ```

2. **Service changes**:
   ```bash
   cd services/oracle
   go run cmd/main.go --btc-host localhost:8332
   ```

3. **SDK usage**:
   ```go
   client := vscdex.NewClient(vscdex.Config{
       Endpoint: "http://localhost:4000",
       Contracts: vscdex.ContractAddresses{
           BtcMapping: "your-btc-mapping-contract-id",
       },
   })

   // Register deposit proof
   amount, err := client.ProveBtcDeposit(ctx, proofBytes)
   ```

## Configuration

Services can be configured via command-line flags or environment variables:

- `VSC_ENDPOINT`: VSC GraphQL endpoint
- `BTC_RPC_HOST`: Bitcoin RPC host:port
- `BTC_RPC_USER`: Bitcoin RPC username
- `BTC_RPC_PASS`: Bitcoin RPC password

## Troubleshooting

### Contract Deployment Issues
- Ensure VSC node is running and accessible
- Check that you have sufficient RC for contract deployment
- Verify contract compilation succeeded (`make contracts`)

### Oracle Connection Issues
- Ensure Bitcoin node is running with RPC enabled
- Check RPC credentials and network connectivity
- Verify Bitcoin node is synced

### Service Communication Issues
- Confirm VSC GraphQL WebSocket endpoint is accessible
- Check service logs for connection errors
- Verify contract IDs are correctly configured
