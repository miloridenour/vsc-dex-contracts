.PHONY: test build clean contracts services sdk

# Test all components
test:
	cd contracts/btc-mapping && tinygo test -v ./...
	cd contracts/token-registry && tinygo test -v ./...
	go test -v ./services/...
	go test -v ./sdk/go/...
	cd sdk/ts && npm test

# Build all services
build: services sdk

# Build service binaries
services:
	cd services/oracle && go build -o ../../bin/oracle ./cmd
	cd services/router && go build -o ../../bin/router ./cmd
	cd services/indexer && go build -o ../../bin/indexer ./cmd

# Build SDK libraries
sdk:
	cd sdk/go && go build ./...
	cd sdk/ts && npm run build

# Build contracts
contracts:
	cd contracts/btc-mapping && tinygo build -o ../../bin/btc-mapping.wasm -target wasm main.go
	cd contracts/token-registry && tinygo build -o ../../bin/token-registry.wasm -target wasm main.go

# Clean build artifacts
clean:
	rm -rf bin/
	cd sdk/ts && npm run clean

# Install development dependencies
setup:
	go mod download
	cd sdk/ts && npm install
	# Install tinygo if not present
	which tinygo || echo "Please install TinyGo: https://tinygo.org/getting-started/install/"

# Run E2E tests
e2e:
	cd e2e && go test -v -timeout 10m ./...

# Format code
fmt:
	go fmt ./...
	cd sdk/ts && npm run fmt


