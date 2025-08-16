package services

import (
	"fmt"
	"sync"
	"time"

	"solana-balance-api/internal/config"
	"solana-balance-api/internal/models"
	"solana-balance-api/pkg/cache"
	"solana-balance-api/pkg/logger"
	"solana-balance-api/pkg/metrics"
	"solana-balance-api/pkg/mutex"

	"go.uber.org/zap"
)

// BalanceService integrates caching, concurrency control, and RPC client
type BalanceService struct {
	rpcClient    SolanaServiceInterface
	cache        *cache.Cache
	requestMutex *mutex.RequestMutex
	config       *config.Config
	metrics      *metrics.MetricsCollector
}

// NewBalanceService creates a new BalanceService instance
func NewBalanceService(rpcClient SolanaServiceInterface, cfg *config.Config) *BalanceService {
	return &BalanceService{
		rpcClient:    rpcClient,
		cache:        cache.New(cfg.Cache.TTL),
		requestMutex: mutex.New(cfg.Cache.CleanupInterval),
		config:       cfg,
		metrics:      metrics.NewMetricsCollector(),
	}
}

// GetBalances fetches balances for multiple wallet addresses with caching and concurrency control
func (bs *BalanceService) GetBalances(addresses []string) (*models.BalanceResponse, error) {
	startTime := time.Now()
	bs.metrics.RecordRequest()

	log := logger.GetLogger()

	if len(addresses) == 0 {
		log.Debug("Empty addresses array provided")
		bs.metrics.RecordRequestComplete(time.Since(startTime), true)
		return &models.BalanceResponse{
			Balances: []models.WalletBalance{},
			Cached:   false,
		}, nil
	}

	log.Info("Processing balance request for multiple addresses",
		zap.Int("address_count", len(addresses)),
	)

	balances := make([]models.WalletBalance, len(addresses))
	allCached := true
	var mu sync.Mutex // Protect allCached variable

	// Use a wait group to handle concurrent processing
	var wg sync.WaitGroup

	for i, address := range addresses {
		wg.Add(1)
		go func(index int, addr string) {
			defer wg.Done()

			walletBalance, cached := bs.getBalanceWithCache(addr)
			balances[index] = *walletBalance

			if !cached {
				mu.Lock()
				allCached = false
				mu.Unlock()
			}
		}(i, address)
	}

	wg.Wait()

	success := true
	for _, balance := range balances {
		if balance.Error != "" {
			success = false
			break
		}
	}

	bs.metrics.RecordRequestComplete(time.Since(startTime), success)

	log.Info("Completed balance request for multiple addresses",
		zap.Int("address_count", len(addresses)),
		zap.Bool("all_cached", allCached),
		zap.Duration("duration", time.Since(startTime)),
	)

	return &models.BalanceResponse{
		Balances: balances,
		Cached:   allCached,
	}, nil
}

// GetBalance fetches balance for a single wallet address
func (bs *BalanceService) GetBalance(address string) (*models.WalletBalance, error) {
	walletBalance, _ := bs.getBalanceWithCache(address)
	return walletBalance, nil
}

// getBalanceWithCache handles the core logic for fetching balance with caching and mutex control
func (bs *BalanceService) getBalanceWithCache(address string) (*models.WalletBalance, bool) {
	log := logger.GetLogger().WithFields(map[string]interface{}{
		"wallet_address": address,
		"component":      "balance_service",
	})

	// First, check if we have a cached result
	if cachedBalance, found := bs.cache.Get(address); found {
		log.Debug("Cache hit for wallet balance")
		bs.metrics.RecordCacheHit()
		return &models.WalletBalance{
			Address: address,
			Balance: cachedBalance,
		}, true
	}

	log.Debug("Cache miss, acquiring mutex for wallet")
	bs.metrics.RecordCacheMiss()

	// Use mutex to prevent duplicate concurrent requests for the same address
	mutexStartTime := time.Now()
	addressMutex := bs.requestMutex.GetMutex(address)
	addressMutex.Lock()
	defer addressMutex.Unlock()

	// Record mutex wait time if it took longer than 1ms
	if time.Since(mutexStartTime) > time.Millisecond {
		bs.metrics.RecordMutexWait()
	}

	// Double-check cache after acquiring mutex (another goroutine might have fetched it)
	if cachedBalance, found := bs.cache.Get(address); found {
		log.Debug("Cache hit after mutex acquisition (populated by concurrent request)")
		bs.metrics.RecordCacheHit()
		return &models.WalletBalance{
			Address: address,
			Balance: cachedBalance,
		}, true
	}

	log.Debug("Fetching balance from RPC client")

	// Fetch from RPC client
	rpcStartTime := time.Now()
	balance, err := bs.rpcClient.GetBalance(address)
	rpcDuration := time.Since(rpcStartTime)

	bs.metrics.RecordRPCCall(rpcDuration, err == nil)

	if err != nil {
		log.Error("Failed to fetch balance from RPC client",
			zap.Error(err),
			zap.Duration("rpc_duration", rpcDuration),
		)
		return &models.WalletBalance{
			Address: address,
			Balance: 0,
			Error:   fmt.Sprintf("Failed to fetch balance: %v", err),
		}, false
	}

	log.Debug("Successfully fetched balance from RPC, caching result",
		zap.Float64("balance", balance),
		zap.Duration("rpc_duration", rpcDuration),
	)

	// Cache the result
	bs.cache.Set(address, balance)

	return &models.WalletBalance{
		Address: address,
		Balance: balance,
	}, false
}

// GetCacheStats returns cache statistics for monitoring
func (bs *BalanceService) GetCacheStats() map[string]interface{} {
	return map[string]interface{}{
		"cache_size":   bs.cache.Size(),
		"mutex_count":  bs.requestMutex.Size(),
		"cache_ttl_ms": bs.config.Cache.TTL.Milliseconds(),
	}
}

// GetMetrics returns performance metrics
func (bs *BalanceService) GetMetrics() *metrics.Metrics {
	return bs.metrics.GetMetrics()
}

// GetPerformanceStats returns comprehensive performance statistics
func (bs *BalanceService) GetPerformanceStats() map[string]interface{} {
	metrics := bs.metrics.GetMetrics()

	return map[string]interface{}{
		"uptime":                   bs.metrics.GetUptime().String(),
		"total_requests":           metrics.TotalRequests,
		"successful_requests":      metrics.SuccessfulRequests,
		"failed_requests":          metrics.FailedRequests,
		"success_rate_percent":     bs.metrics.GetSuccessRate(),
		"average_response_time_ms": metrics.AverageResponseTime.Milliseconds(),
		"min_response_time_ms":     metrics.MinResponseTime.Milliseconds(),
		"max_response_time_ms":     metrics.MaxResponseTime.Milliseconds(),
		"cache_hits":               metrics.CacheHits,
		"cache_misses":             metrics.CacheMisses,
		"cache_hit_ratio_percent":  bs.metrics.GetCacheHitRatio(),
		"rpc_calls":                metrics.RPCCalls,
		"rpc_failures":             metrics.RPCFailures,
		"average_rpc_time_ms":      metrics.AverageRPCTime.Milliseconds(),
		"active_requests":          metrics.ActiveRequests,
		"mutex_waits":              metrics.MutexWaits,
		"cache_size":               bs.cache.Size(),
		"mutex_count":              bs.requestMutex.Size(),
	}
}

// ClearCache clears all cached entries
func (bs *BalanceService) ClearCache() {
	bs.cache.Clear()
}

// Stop gracefully shuts down the service
func (bs *BalanceService) Stop() {
	bs.cache.Stop()
	bs.requestMutex.Stop()
}

// GetMetricsCollector returns the metrics collector for middleware integration
func (bs *BalanceService) GetMetricsCollector() *metrics.MetricsCollector {
	return bs.metrics
}
