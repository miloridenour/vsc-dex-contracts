# VSC DEX Mapping

A modular, external DEX mapping system for VSC blockchain that enables seamless cross-chain asset swaps through UTXO mapping and automated liquidity routing.

## Overview

VSC DEX Mapping provides a complete infrastructure for decentralized exchange operations with support for cross-chain assets, automated routing, and real-time indexing. Built as a collection of microservices that integrate with VSC through public APIs.

## Features

- **Cross-Chain Asset Mapping**: UTXO-based asset mapping with SPV verification
- **Automated DEX Routing**: Intelligent route planning with multi-hop support
- **Real-Time Indexing**: Event-driven indexing and query APIs
- **Extensible Architecture**: Plugin-based design for new blockchains
- **Multi-Language SDKs**: Go and TypeScript client libraries

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│ External        │    │   VSC Node      │    │   DEX Frontend  │
│ Blockchains     │◄──►│   (Core)        │◄──►│   Applications  │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         ▲                        ▲                        ▲
         │                        │                        │
    ┌────▼────┐              ┌────▼────┐              ┌────▼────┐
    │ Oracles │              │ Smart   │              │ Route   │
    │ Service │─────────────►│Contracts│◄─────────────│Planner  │
    └─────────┘              └─────────┘              └─────────┘
                                   ▲                        ▲
                                   │                        │
                              ┌────▼────┐              ┌────▼────┐
                              │ Indexer │              │  SDK    │
                              │ Service │◄─────────────┤ Libraries│
                              └─────────┘              └─────────┘
```

## Components

### Core Services
- **Oracle Services**: Cross-chain data relays and proof verification
- **DEX Router**: Automated swap routing and transaction composition
- **Indexer**: Real-time event processing and query APIs

### Smart Contracts
- **Mapping Contracts**: UTXO and asset mapping logic
- **Token Registry**: Wrapped asset management and metadata

### Development Tools
- **Go SDK**: Backend integration libraries
- **TypeScript SDK**: Frontend application support
- **CLI Tools**: Deployment and administration utilities

## Quick Start

```bash
# Clone the repository
git clone <repository-url>
cd vsc-dex-mapping

# Install dependencies
make setup

# Run tests
make test

# Build all components
make build
```

## Project Structure

```
├── contracts/          # Smart contracts (TinyGo)
├── services/           # Microservices (Go)
├── sdk/               # Client libraries (Go/TypeScript)
├── cli/               # Command-line tools
├── docs/              # Documentation
├── e2e/               # End-to-end tests
└── scripts/           # Build and deployment scripts
```

## Development

### Prerequisites

- Go 1.21+
- TinyGo (for contracts)
- Node.js 18+ (for TypeScript SDK)

### Testing

```bash
# Run all tests
make test

# Run specific test suites
go test ./services/router/...
go test ./e2e/...

# Run with coverage
go test -cover ./...
```

### Building

```bash
# Build all components
make build

# Build individual services
cd services/router && go build
cd contracts/btc-mapping && tinygo build -target wasm
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes with tests
4. Submit a pull request

## License

MIT License - see LICENSE file for details
