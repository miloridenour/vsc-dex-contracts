.PHONY: test build clean contracts services sdk

# Test all components
test:
	cd contracts/btc-mapping && tinygo test -v ./...
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

# Run unit tests
test:
	cd contracts/dex-router/test && go test -v ./...

# Run unit tests with coverage
test-cover:
	cd contracts/dex-router/test && go test -cover -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run E2E tests
e2e:
	./test-dex-e2e.sh

# Run demo
demo:
	node demo-dex.js

# Format code
fmt:
	go fmt ./...
	cd sdk/ts && npm run fmt



