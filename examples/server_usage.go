package examples

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

// Example demonstrating how to interact with the Solana Balance API server
func ServerUsageExample() {
	// Server configuration
	serverURL := "http://localhost:8080"
	apiKey := "your-api-key-here" // Replace with actual API key

	// Example wallet addresses (these are example addresses)
	wallets := []string{
		"11111111111111111111111111111112",            // System program
		"TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA", // Token program
	}

	fmt.Println("Solana Balance API Server Usage Example")
	fmt.Println("======================================")

	// 1. Check server health
	fmt.Println("\n1. Checking server health...")
	if err := checkHealth(serverURL); err != nil {
		log.Printf("Health check failed: %v", err)
		return
	}
	fmt.Println("✓ Server is healthy")

	// 2. Get balances for multiple wallets
	fmt.Println("\n2. Getting balances for multiple wallets...")
	balances, err := getBalances(serverURL, apiKey, wallets)
	if err != nil {
		log.Printf("Failed to get balances: %v", err)
		return
	}

	fmt.Printf("✓ Retrieved balances for %d wallets:\n", len(balances.Balances))
	for _, balance := range balances.Balances {
		if balance.Error != "" {
			fmt.Printf("  - %s: ERROR - %s\n", balance.Address, balance.Error)
		} else {
			fmt.Printf("  - %s: %.9f SOL\n", balance.Address, balance.Balance)
		}
	}
	fmt.Printf("  Cached: %v\n", balances.Cached)

	// 3. Demonstrate caching by making the same request again
	fmt.Println("\n3. Making the same request again to demonstrate caching...")
	time.Sleep(100 * time.Millisecond) // Small delay
	balances2, err := getBalances(serverURL, apiKey, wallets)
	if err != nil {
		log.Printf("Failed to get balances (second request): %v", err)
		return
	}
	fmt.Printf("✓ Second request completed. Cached: %v\n", balances2.Cached)

	// 4. Check server status
	fmt.Println("\n4. Checking server status...")
	if err := checkStatus(serverURL); err != nil {
		log.Printf("Status check failed: %v", err)
		return
	}
	fmt.Println("✓ Server status retrieved")

	fmt.Println("\n✓ All examples completed successfully!")
	fmt.Println("\nNote: This example requires:")
	fmt.Println("- The server to be running on localhost:8080")
	fmt.Println("- A valid API key in MongoDB")
	fmt.Println("- MongoDB running and accessible")
}

// BalanceRequest represents the request structure
type BalanceRequest struct {
	Wallets []string `json:"wallets"`
}

// BalanceResponse represents the response structure
type BalanceResponse struct {
	Balances []WalletBalance `json:"balances"`
	Cached   bool            `json:"cached"`
}

// WalletBalance represents individual wallet balance
type WalletBalance struct {
	Address string  `json:"address"`
	Balance float64 `json:"balance"`
	Error   string  `json:"error,omitempty"`
}

// checkHealth performs a health check on the server
func checkHealth(serverURL string) error {
	resp, err := http.Get(serverURL + "/health")
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed with status: %d", resp.StatusCode)
	}

	var health map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		return fmt.Errorf("failed to decode health response: %w", err)
	}

	fmt.Printf("  Status: %v\n", health["status"])
	fmt.Printf("  Service: %v\n", health["service"])

	return nil
}

// checkStatus gets detailed server status
func checkStatus(serverURL string) error {
	resp, err := http.Get(serverURL + "/status")
	if err != nil {
		return fmt.Errorf("failed to get server status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status check failed with status: %d", resp.StatusCode)
	}

	var status map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return fmt.Errorf("failed to decode status response: %w", err)
	}

	fmt.Printf("  Service: %v\n", status["service"])
	fmt.Printf("  Status: %v\n", status["status"])
	fmt.Printf("  RPC Healthy: %v\n", status["rpc_healthy"])
	fmt.Printf("  Uptime: %v\n", status["uptime"])
	fmt.Printf("  Version: %v\n", status["version"])

	return nil
}

// getBalances fetches balances for the given wallet addresses
func getBalances(serverURL, apiKey string, wallets []string) (*BalanceResponse, error) {
	// Prepare request
	request := BalanceRequest{
		Wallets: wallets,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", serverURL+"/api/get-balance", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", apiKey)

	// Make request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var balanceResp BalanceResponse
	if err := json.Unmarshal(body, &balanceResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Print rate limit headers if available
	if limit := resp.Header.Get("X-RateLimit-Limit"); limit != "" {
		fmt.Printf("  Rate Limit: %s requests per minute\n", limit)
		fmt.Printf("  Remaining: %s\n", resp.Header.Get("X-RateLimit-Remaining"))
	}

	return &balanceResp, nil
}

// Example of how to run the server programmatically (for testing)
func runServerExample() {
	fmt.Println("To run the server, use:")
	fmt.Println("  go run ./cmd/server")
	fmt.Println("")
	fmt.Println("Environment variables you can set:")
	fmt.Println("  SERVER_PORT=8080")
	fmt.Println("  SERVER_HOST=0.0.0.0")
	fmt.Println("  MONGODB_URI=mongodb://localhost:27017")
	fmt.Println("  MONGODB_DATABASE=solana_api")
	fmt.Println("  SOLANA_RPC_ENDPOINT=https://your-helius-endpoint")
	fmt.Println("  RATE_LIMIT_REQUESTS_PER_MINUTE=10")
	fmt.Println("  CACHE_TTL=10s")

	// Example of setting environment variables programmatically
	os.Setenv("SERVER_PORT", "8080")
	os.Setenv("GIN_MODE", "release")
}
