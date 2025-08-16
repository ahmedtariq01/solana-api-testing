package examples

import (
	"fmt"
	"log"
	"time"

	"solana-balance-api/internal/config"
	"solana-balance-api/internal/services"
)

func SolanaUsageExample() {
	// Load configuration
	cfg := config.LoadConfig()

	// Create Solana client
	solanaClient := services.NewSolanaClient(&cfg.RPC)

	// Test health check
	fmt.Println("Checking RPC health...")
	if err := solanaClient.IsHealthy(); err != nil {
		log.Printf("RPC health check failed: %v", err)
	} else {
		fmt.Println("RPC is healthy!")
	}

	// Example wallet addresses (these are well-known program addresses)
	addresses := []string{
		"11111111111111111111111111111112",            // System Program
		"TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA", // Token Program
		"So11111111111111111111111111111111111111112", // Wrapped SOL
	}

	// Get single balance
	fmt.Printf("\nGetting balance for single address: %s\n", addresses[0])
	balance, err := solanaClient.GetBalance(addresses[0])
	if err != nil {
		log.Printf("Error getting balance: %v", err)
	} else {
		fmt.Printf("Balance: %.9f SOL\n", balance)
	}

	// Get multiple balances
	fmt.Printf("\nGetting balances for multiple addresses...\n")
	balances, err := solanaClient.GetBalances(addresses)
	if err != nil {
		log.Printf("Error getting balances: %v", err)
	} else {
		for address, balance := range balances {
			fmt.Printf("Address: %s, Balance: %.9f SOL\n", address, balance)
		}
	}

	// Demonstrate timeout handling
	fmt.Printf("\nTesting with short timeout...\n")
	shortTimeoutConfig := &config.RPCConfig{
		Endpoint: cfg.RPC.Endpoint,
		Timeout:  1 * time.Millisecond, // Very short timeout
		APIKey:   cfg.RPC.APIKey,
	}

	shortTimeoutClient := services.NewSolanaClient(shortTimeoutConfig)
	_, err = shortTimeoutClient.GetBalance(addresses[0])
	if err != nil {
		fmt.Printf("Expected timeout error: %v\n", err)
	}

	fmt.Println("\nSolana client example completed!")
}
