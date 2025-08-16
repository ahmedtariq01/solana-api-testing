package services

import "solana-balance-api/internal/models"

// AuthServiceInterface defines the interface for authentication services
type AuthServiceInterface interface {
	ValidateAPIKey(key string) (*models.APIKey, error)
}

// SolanaServiceInterface defines the interface for Solana RPC operations
type SolanaServiceInterface interface {
	GetBalance(address string) (float64, error)
	GetBalances(addresses []string) (map[string]float64, error)
}

// BalanceServiceInterface defines the interface for balance operations
type BalanceServiceInterface interface {
	GetBalances(addresses []string) (*models.BalanceResponse, error)
	GetBalance(address string) (*models.WalletBalance, error)
}
