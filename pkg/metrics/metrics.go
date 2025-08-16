package metrics

import (
	"sync"
	"sync/atomic"
	"time"
)

// Metrics holds performance metrics for the application
type Metrics struct {
	// Request metrics
	TotalRequests      int64 `json:"total_requests"`
	SuccessfulRequests int64 `json:"successful_requests"`
	FailedRequests     int64 `json:"failed_requests"`

	// Response time metrics
	AverageResponseTime time.Duration `json:"average_response_time"`
	MinResponseTime     time.Duration `json:"min_response_time"`
	MaxResponseTime     time.Duration `json:"max_response_time"`

	// Cache metrics
	CacheHits   int64 `json:"cache_hits"`
	CacheMisses int64 `json:"cache_misses"`

	// RPC metrics
	RPCCalls       int64         `json:"rpc_calls"`
	RPCFailures    int64         `json:"rpc_failures"`
	AverageRPCTime time.Duration `json:"average_rpc_time"`

	// Concurrency metrics
	ActiveRequests int64 `json:"active_requests"`
	MutexWaits     int64 `json:"mutex_waits"`

	// Internal fields for calculations
	totalResponseTime time.Duration
	totalRPCTime      time.Duration
	mutex             sync.RWMutex
}

// MetricsCollector provides thread-safe metrics collection
type MetricsCollector struct {
	metrics   *Metrics
	startTime time.Time
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		metrics: &Metrics{
			MinResponseTime: time.Duration(^uint64(0) >> 1), // Max duration
		},
		startTime: time.Now(),
	}
}

// RecordRequest records a new request
func (mc *MetricsCollector) RecordRequest() {
	atomic.AddInt64(&mc.metrics.TotalRequests, 1)
	atomic.AddInt64(&mc.metrics.ActiveRequests, 1)
}

// RecordRequestComplete records request completion
func (mc *MetricsCollector) RecordRequestComplete(duration time.Duration, success bool) {
	atomic.AddInt64(&mc.metrics.ActiveRequests, -1)

	if success {
		atomic.AddInt64(&mc.metrics.SuccessfulRequests, 1)
	} else {
		atomic.AddInt64(&mc.metrics.FailedRequests, 1)
	}

	// Update response time metrics
	mc.metrics.mutex.Lock()
	defer mc.metrics.mutex.Unlock()

	mc.metrics.totalResponseTime += duration

	if duration < mc.metrics.MinResponseTime {
		mc.metrics.MinResponseTime = duration
	}

	if duration > mc.metrics.MaxResponseTime {
		mc.metrics.MaxResponseTime = duration
	}

	// Calculate average
	totalRequests := atomic.LoadInt64(&mc.metrics.TotalRequests)
	if totalRequests > 0 {
		mc.metrics.AverageResponseTime = mc.metrics.totalResponseTime / time.Duration(totalRequests)
	}
}

// RecordCacheHit records a cache hit
func (mc *MetricsCollector) RecordCacheHit() {
	atomic.AddInt64(&mc.metrics.CacheHits, 1)
}

// RecordCacheMiss records a cache miss
func (mc *MetricsCollector) RecordCacheMiss() {
	atomic.AddInt64(&mc.metrics.CacheMisses, 1)
}

// RecordRPCCall records an RPC call
func (mc *MetricsCollector) RecordRPCCall(duration time.Duration, success bool) {
	atomic.AddInt64(&mc.metrics.RPCCalls, 1)

	if !success {
		atomic.AddInt64(&mc.metrics.RPCFailures, 1)
	}

	// Update RPC time metrics
	mc.metrics.mutex.Lock()
	defer mc.metrics.mutex.Unlock()

	mc.metrics.totalRPCTime += duration

	// Calculate average
	totalRPCCalls := atomic.LoadInt64(&mc.metrics.RPCCalls)
	if totalRPCCalls > 0 {
		mc.metrics.AverageRPCTime = mc.metrics.totalRPCTime / time.Duration(totalRPCCalls)
	}
}

// RecordMutexWait records a mutex wait
func (mc *MetricsCollector) RecordMutexWait() {
	atomic.AddInt64(&mc.metrics.MutexWaits, 1)
}

// GetMetrics returns a copy of current metrics
func (mc *MetricsCollector) GetMetrics() *Metrics {
	mc.metrics.mutex.RLock()
	defer mc.metrics.mutex.RUnlock()

	// Create a copy to avoid race conditions
	return &Metrics{
		TotalRequests:       atomic.LoadInt64(&mc.metrics.TotalRequests),
		SuccessfulRequests:  atomic.LoadInt64(&mc.metrics.SuccessfulRequests),
		FailedRequests:      atomic.LoadInt64(&mc.metrics.FailedRequests),
		AverageResponseTime: mc.metrics.AverageResponseTime,
		MinResponseTime:     mc.metrics.MinResponseTime,
		MaxResponseTime:     mc.metrics.MaxResponseTime,
		CacheHits:           atomic.LoadInt64(&mc.metrics.CacheHits),
		CacheMisses:         atomic.LoadInt64(&mc.metrics.CacheMisses),
		RPCCalls:            atomic.LoadInt64(&mc.metrics.RPCCalls),
		RPCFailures:         atomic.LoadInt64(&mc.metrics.RPCFailures),
		AverageRPCTime:      mc.metrics.AverageRPCTime,
		ActiveRequests:      atomic.LoadInt64(&mc.metrics.ActiveRequests),
		MutexWaits:          atomic.LoadInt64(&mc.metrics.MutexWaits),
	}
}

// GetUptime returns the uptime since metrics collection started
func (mc *MetricsCollector) GetUptime() time.Duration {
	return time.Since(mc.startTime)
}

// Reset resets all metrics
func (mc *MetricsCollector) Reset() {
	mc.metrics.mutex.Lock()
	defer mc.metrics.mutex.Unlock()

	atomic.StoreInt64(&mc.metrics.TotalRequests, 0)
	atomic.StoreInt64(&mc.metrics.SuccessfulRequests, 0)
	atomic.StoreInt64(&mc.metrics.FailedRequests, 0)
	atomic.StoreInt64(&mc.metrics.CacheHits, 0)
	atomic.StoreInt64(&mc.metrics.CacheMisses, 0)
	atomic.StoreInt64(&mc.metrics.RPCCalls, 0)
	atomic.StoreInt64(&mc.metrics.RPCFailures, 0)
	atomic.StoreInt64(&mc.metrics.ActiveRequests, 0)
	atomic.StoreInt64(&mc.metrics.MutexWaits, 0)

	mc.metrics.AverageResponseTime = 0
	mc.metrics.MinResponseTime = time.Duration(^uint64(0) >> 1)
	mc.metrics.MaxResponseTime = 0
	mc.metrics.AverageRPCTime = 0
	mc.metrics.totalResponseTime = 0
	mc.metrics.totalRPCTime = 0

	mc.startTime = time.Now()
}

// GetCacheHitRatio returns the cache hit ratio as a percentage
func (mc *MetricsCollector) GetCacheHitRatio() float64 {
	hits := atomic.LoadInt64(&mc.metrics.CacheHits)
	misses := atomic.LoadInt64(&mc.metrics.CacheMisses)
	total := hits + misses

	if total == 0 {
		return 0.0
	}

	return float64(hits) / float64(total) * 100.0
}

// GetSuccessRate returns the success rate as a percentage
func (mc *MetricsCollector) GetSuccessRate() float64 {
	successful := atomic.LoadInt64(&mc.metrics.SuccessfulRequests)
	total := atomic.LoadInt64(&mc.metrics.TotalRequests)

	if total == 0 {
		return 0.0
	}

	return float64(successful) / float64(total) * 100.0
}
