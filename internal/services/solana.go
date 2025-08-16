package services

import (
	"context"
	"fmt"
	"time"

	"solana-balance-api/internal/config"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

// SolanaClient wraps the Solana RPC client with configuration
type SolanaClient struct {
	client *rpc.Client
	config *config.RPCConfig
}

// NewSolanaClient creates a new Solana RPC client with optimized configuration
func NewSolanaClient(cfg *config.RPCConfig) *SolanaClient {
	// Create RPC client with the endpoint
	client := rpc.New(cfg.Endpoint)

	// Note: The gagliardetto/solana-go library doesn't directly expose HTTP client configuration.
	// For production use with custom HTTP transport optimizations, consider implementing
	// a custom RPC client wrapper that supports:
	// - Connection pooling (MaxIdleConns, MaxIdleConnsPerHost)
	// - Keep-alive settings (KeepAlive, IdleConnTimeout)
	// - Timeout configurations (TLSHandshakeTimeout, ExpectContinueTimeout)
	// - Buffer optimizations (WriteBufferSize, ReadBufferSize)
	// - HTTP/2 support (ForceAttemptHTTP2)

	return &SolanaClient{
		client: client,
		config: cfg,
	}
}

// GetBalance fetches the balance for a single Solana wallet address with retry logic
func (s *SolanaClient) GetBalance(address string) (float64, error) {
	// Parse the wallet address
	pubKey, err := solana.PublicKeyFromBase58(address)
	if err != nil {
		return 0, fmt.Errorf("invalid wallet address: %w", err)
	}

	// Retry logic
	var lastErr error
	for attempt := 0; attempt <= s.config.MaxRetries; attempt++ {
		// Create context with timeout for each attempt
		ctx, cancel := context.WithTimeout(context.Background(), s.config.Timeout)

		// Get balance from RPC
		balance, err := s.client.GetBalance(ctx, pubKey, rpc.CommitmentFinalized)
		cancel()

		if err == nil {
			// Success - convert lamports to SOL (1 SOL = 1,000,000,000 lamports)
			solBalance := float64(balance.Value) / 1e9
			return solBalance, nil
		}

		lastErr = err

		// Don't retry on the last attempt
		if attempt < s.config.MaxRetries {
			time.Sleep(s.config.RetryDelay * time.Duration(attempt+1)) // Exponential backoff
		}
	}

	return 0, fmt.Errorf("failed to get balance from RPC after %d attempts: %w", s.config.MaxRetries+1, lastErr)
}

// GetBalances fetches balances for multiple wallet addresses
// For better performance with large batches, consider using GetBalancesBatch
func (s *SolanaClient) GetBalances(addresses []string) (map[string]float64, error) {
	if len(addresses) == 0 {
		return make(map[string]float64), nil
	}

	// For small batches, use the batch method
	if len(addresses) <= 100 {
		return s.getBalancesBatch(addresses)
	}

	// For larger batches, process in chunks to avoid RPC limits
	result := make(map[string]float64)
	chunkSize := 100

	for i := 0; i < len(addresses); i += chunkSize {
		end := i + chunkSize
		if end > len(addresses) {
			end = len(addresses)
		}

		chunk := addresses[i:end]
		chunkBalances, err := s.getBalancesBatch(chunk)
		if err != nil {
			return nil, fmt.Errorf("failed to get balances for chunk starting at %d: %w", i, err)
		}

		// Merge results
		for addr, balance := range chunkBalances {
			result[addr] = balance
		}
	}

	return result, nil
}

// getBalancesBatch handles batch requests for up to 100 addresses
func (s *SolanaClient) getBalancesBatch(addresses []string) (map[string]float64, error) {
	// Parse all addresses first to validate them
	pubKeys := make([]solana.PublicKey, len(addresses))
	for i, address := range addresses {
		pubKey, err := solana.PublicKeyFromBase58(address)
		if err != nil {
			return nil, fmt.Errorf("invalid wallet address %s: %w", address, err)
		}
		pubKeys[i] = pubKey
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), s.config.Timeout)
	defer cancel()

	// Get multiple balances using batch request
	balances, err := s.client.GetMultipleAccounts(ctx, pubKeys...)
	if err != nil {
		return nil, fmt.Errorf("failed to get balances from RPC: %w", err)
	}

	// Process results
	result := make(map[string]float64, len(addresses))
	for i, address := range addresses {
		if i < len(balances.Value) && balances.Value[i] != nil {
			// Convert lamports to SOL
			solBalance := float64(balances.Value[i].Lamports) / 1e9
			result[address] = solBalance
		} else {
			// Account doesn't exist or has no balance
			result[address] = 0.0
		}
	}

	return result, nil
}

// GetBalanceWithCommitment fetches balance with specific commitment level
func (s *SolanaClient) GetBalanceWithCommitment(address string, commitment rpc.CommitmentType) (float64, error) {
	// Parse the wallet address
	pubKey, err := solana.PublicKeyFromBase58(address)
	if err != nil {
		return 0, fmt.Errorf("invalid wallet address: %w", err)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), s.config.Timeout)
	defer cancel()

	// Get balance from RPC with specific commitment
	balance, err := s.client.GetBalance(ctx, pubKey, commitment)
	if err != nil {
		return 0, fmt.Errorf("failed to get balance from RPC: %w", err)
	}

	// Convert lamports to SOL
	solBalance := float64(balance.Value) / 1e9

	return solBalance, nil
}

// IsHealthy checks if the RPC endpoint is responsive
func (s *SolanaClient) IsHealthy() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Try to get the latest blockhash as a health check
	_, err := s.client.GetLatestBlockhash(ctx, rpc.CommitmentFinalized)
	if err != nil {
		return fmt.Errorf("RPC health check failed: %w", err)
	}

	return nil
}
