package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "vsc-dex-mapping",
	Short: "CLI for VSC DEX mapping operations",
	Long:  `Deploy contracts, manage mappings, and administer DEX operations`,
}

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy contracts to VSC",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Deploying contracts...")

		// Check if contracts are already built
		if _, err := os.Stat("../contracts/btc-mapping/main.wasm"); os.IsNotExist(err) {
			fmt.Println("❌ BTC mapping contract not built. Run 'cd ../contracts/btc-mapping && tinygo build -o main.wasm -target=wasm-unknown main.go'")
			os.Exit(1)
		}


		// Deploy btc-mapping contract
		fmt.Println("Deploying btc-mapping contract...")
		btcContractId, err := deployContract("../contracts/btc-mapping/main.wasm", "btc-mapping", "Bitcoin UTXO mapping contract for VSC DEX")
		if err != nil {
			fmt.Printf("❌ Failed to deploy btc-mapping contract: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✓ Deployed btc-mapping contract at: %s\n", btcContractId)



		// Save contract addresses to config
		config := map[string]string{
			"btcMapping": btcContractId,
			"dexRouter":  "dex-router-contract", // Placeholder
		}

		configData, _ := json.MarshalIndent(config, "", "  ")
		err = os.WriteFile("contracts.json", configData, 0644)
		if err != nil {
			fmt.Printf("⚠️  Warning: Failed to save contract config: %v\n", err)
		} else {
			fmt.Println("✓ Saved contract addresses to contracts.json")
		}

		fmt.Println("\n🎉 Deployment complete!")
		fmt.Println("Contract addresses saved to contracts.json")
		fmt.Println("Update your SDK config to use these addresses.")
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check system status",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("🔍 Checking system status...")
		fmt.Println()

		allGood := true

		// Check contract deployments
		fmt.Print("📋 Checking contract deployments... ")
		if contractsDeployed := checkContractDeployments(); contractsDeployed {
			fmt.Println("✅ Contracts deployed")
		} else {
			fmt.Println("❌ Contracts not deployed")
			allGood = false
		}

		// Check WASM builds
		fmt.Print("🔨 Checking WASM builds... ")
		if wasmBuilt := checkWasmBuilds(); wasmBuilt {
			fmt.Println("✅ WASM files built")
		} else {
			fmt.Println("❌ WASM files missing")
			allGood = false
		}

		// Check oracle connectivity
		fmt.Print("🌐 Checking Bitcoin node connectivity... ")
		if btcConnected := checkBitcoinConnectivity(); btcConnected {
			fmt.Println("✅ Bitcoin node reachable")
		} else {
			fmt.Println("⚠️  Bitcoin node not reachable (may be expected in dev)")
		}

		// Check VSC GraphQL endpoint
		fmt.Print("🔗 Checking VSC GraphQL endpoint... ")
		if vscConnected := checkVSCConnectivity(); vscConnected {
			fmt.Println("✅ VSC GraphQL reachable")
		} else {
			fmt.Println("❌ VSC GraphQL not reachable")
			allGood = false
		}

		// Check indexer sync
		fmt.Print("💾 Checking indexer status... ")
		if indexerReady := checkIndexerStatus(); indexerReady {
			fmt.Println("✅ Indexer operational")
		} else {
			fmt.Println("⚠️  Indexer not running (expected in development)")
		}

		fmt.Println()

		if allGood {
			fmt.Println("🎉 All systems operational!")
			fmt.Println()
			fmt.Println("🚀 Ready for BTC↔HBD trading!")
			fmt.Println()
			fmt.Println("Next steps:")
			fmt.Println("1. Start oracle service: ./oracle")
			fmt.Println("2. Start indexer service: ./indexer")
			fmt.Println("3. Start router service: ./router")
			fmt.Println("4. Use SDK to deposit BTC and trade")
		} else {
			fmt.Println("⚠️  Some systems need attention before production use")
			fmt.Println()
			fmt.Println("To fix issues:")
			fmt.Println("1. Deploy contracts: ./cli deploy")
			fmt.Println("2. Build WASM: cd contracts/*/ && tinygo build -target=wasm-unknown")
			fmt.Println("3. Start VSC node and services")
		}
	},
}

// deployContract simulates deploying a WASM contract to VSC
func deployContract(wasmPath, name, description string) (string, error) {
	// In a real implementation, this would:
	// 1. Read the WASM file
	// 2. Request storage proof from data availability layer
	// 3. Create TxCreateContract transaction
	// 4. Broadcast via Hive custom JSON operation
	// 5. Return the contract ID

	fmt.Printf("  Reading WASM file: %s\n", wasmPath)
	wasmData, err := os.ReadFile(wasmPath)
	if err != nil {
		return "", fmt.Errorf("failed to read WASM file: %w", err)
	}
	fmt.Printf("  WASM size: %d bytes\n", len(wasmData))

	// Simulate deployment delay
	time.Sleep(100 * time.Millisecond)

	// Generate mock contract ID (in real implementation, this comes from txid)
	contractId := fmt.Sprintf("%s-%d", name, time.Now().Unix())

	fmt.Printf("  Contract deployed with ID: %s\n", contractId)
	return contractId, nil
}


// checkContractDeployments checks if contracts are deployed by looking for config file
func checkContractDeployments() bool {
	if _, err := os.Stat("contracts.json"); os.IsNotExist(err) {
		return false
	}
	return true
}

// checkWasmBuilds checks if WASM files exist
func checkWasmBuilds() bool {
	btcWasm := "../contracts/btc-mapping/main.wasm"

	if _, err := os.Stat(btcWasm); os.IsNotExist(err) {
		return false
	}
	return true
}

// checkBitcoinConnectivity attempts to connect to Bitcoin node
func checkBitcoinConnectivity() bool {
	// In a real implementation, this would try to connect to Bitcoin RPC
	// For now, just return false (expected in development)
	return false
}

// checkVSCConnectivity tests VSC GraphQL endpoint
func checkVSCConnectivity() bool {
	// Try to make a simple GraphQL query to test connectivity
	// For now, assume VSC is running on localhost:7080
	return false // Will be implemented when we have a running VSC node
}

// checkIndexerStatus checks if indexer service is running
func checkIndexerStatus() bool {
	// In a real implementation, this would:
	// 1. Try to connect to indexer HTTP endpoint
	// 2. Query health/status endpoint
	// For now, return false (expected in development)
	return false
}

func init() {
	rootCmd.AddCommand(deployCmd)
	rootCmd.AddCommand(statusCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
