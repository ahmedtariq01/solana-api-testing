package middleware

import (
	"strconv"
	"time"

	"solana-balance-api/pkg/metrics"

	"github.com/gin-gonic/gin"
)

// PerformanceMiddleware tracks request performance metrics
func PerformanceMiddleware(metricsCollector *metrics.MetricsCollector) gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()

		// Record request start
		metricsCollector.RecordRequest()

		// Process request
		c.Next()

		// Calculate duration
		duration := time.Since(startTime)

		// Determine if request was successful (2xx status codes)
		success := c.Writer.Status() >= 200 && c.Writer.Status() < 300

		// Record request completion
		metricsCollector.RecordRequestComplete(duration, success)

		// Add performance headers
		c.Header("X-Response-Time", duration.String())
		c.Header("X-Response-Time-Ms", strconv.FormatInt(duration.Milliseconds(), 10))
	}
}

// RequestSizeMiddleware tracks request and response sizes
func RequestSizeMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Track request size
		if c.Request.ContentLength > 0 {
			c.Header("X-Request-Size", strconv.FormatInt(c.Request.ContentLength, 10))
		}

		// Process request
		c.Next()

		// Track response size (approximate)
		responseSize := c.Writer.Size()
		if responseSize > 0 {
			c.Header("X-Response-Size", strconv.Itoa(responseSize))
		}
	}
}

// ConcurrencyMiddleware tracks active request count
func ConcurrencyMiddleware(metricsCollector *metrics.MetricsCollector) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Add concurrency tracking header
		activeRequests := metricsCollector.GetMetrics().ActiveRequests
		c.Header("X-Active-Requests", strconv.FormatInt(activeRequests, 10))

		c.Next()
	}
}
