package metrics

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMetricsCollector(t *testing.T) {
	collector := NewMetricsCollector()

	t.Run("InitialState", func(t *testing.T) {
		metrics := collector.GetMetrics()
		assert.Equal(t, int64(0), metrics.TotalRequests)
		assert.Equal(t, int64(0), metrics.SuccessfulRequests)
		assert.Equal(t, int64(0), metrics.FailedRequests)
		assert.Equal(t, int64(0), metrics.CacheHits)
		assert.Equal(t, int64(0), metrics.CacheMisses)
	})

	t.Run("RecordRequest", func(t *testing.T) {
		collector.RecordRequest()
		metrics := collector.GetMetrics()
		assert.Equal(t, int64(1), metrics.TotalRequests)
		assert.Equal(t, int64(1), metrics.ActiveRequests)
	})

	t.Run("RecordRequestComplete", func(t *testing.T) {
		duration := 100 * time.Millisecond
		collector.RecordRequestComplete(duration, true)

		metrics := collector.GetMetrics()
		assert.Equal(t, int64(1), metrics.SuccessfulRequests)
		assert.Equal(t, int64(0), metrics.ActiveRequests)
		assert.Equal(t, duration, metrics.AverageResponseTime)
		assert.Equal(t, duration, metrics.MinResponseTime)
		assert.Equal(t, duration, metrics.MaxResponseTime)
	})

	t.Run("CacheMetrics", func(t *testing.T) {
		collector.RecordCacheHit()
		collector.RecordCacheHit()
		collector.RecordCacheMiss()

		metrics := collector.GetMetrics()
		assert.Equal(t, int64(2), metrics.CacheHits)
		assert.Equal(t, int64(1), metrics.CacheMisses)

		hitRatio := collector.GetCacheHitRatio()
		assert.InDelta(t, 66.67, hitRatio, 0.1)
	})

	t.Run("RPCMetrics", func(t *testing.T) {
		duration := 50 * time.Millisecond
		collector.RecordRPCCall(duration, true)
		collector.RecordRPCCall(duration*2, false)

		metrics := collector.GetMetrics()
		assert.Equal(t, int64(2), metrics.RPCCalls)
		assert.Equal(t, int64(1), metrics.RPCFailures)
		assert.Equal(t, duration*3/2, metrics.AverageRPCTime)
	})

	t.Run("SuccessRate", func(t *testing.T) {
		// Reset for clean test
		collector.Reset()

		collector.RecordRequest()
		collector.RecordRequestComplete(10*time.Millisecond, true)

		collector.RecordRequest()
		collector.RecordRequestComplete(20*time.Millisecond, true)

		collector.RecordRequest()
		collector.RecordRequestComplete(30*time.Millisecond, false)

		successRate := collector.GetSuccessRate()
		assert.InDelta(t, 66.67, successRate, 0.1)
	})

	t.Run("Reset", func(t *testing.T) {
		collector.Reset()

		metrics := collector.GetMetrics()
		assert.Equal(t, int64(0), metrics.TotalRequests)
		assert.Equal(t, int64(0), metrics.SuccessfulRequests)
		assert.Equal(t, int64(0), metrics.CacheHits)
		assert.Equal(t, int64(0), metrics.RPCCalls)
	})
}
