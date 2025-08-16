package models

import "time"

// BalanceRequest represents the incoming request for wallet balances
type BalanceRequest struct {
	Wallets []string `json:"wallets"`
}

// BalanceResponse represents the response containing wallet balances
type BalanceResponse struct {
	Balances []WalletBalance `json:"balances"`
	Cached   bool            `json:"cached"`
}

// WalletBalance represents the balance information for a single wallet
type WalletBalance struct {
	Address string  `json:"address"`
	Balance float64 `json:"balance"`
	Error   string  `json:"error,omitempty"`
}

// CacheEntry represents a cached balance entry with TTL
type CacheEntry struct {
	Balance   float64   `json:"balance"`
	Timestamp time.Time `json:"timestamp"`
}

// RequestCounter tracks request counts for rate limiting
type RequestCounter struct {
	Count     int       `json:"count"`
	ResetTime time.Time `json:"reset_time"`
}
