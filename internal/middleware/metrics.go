package middleware

import (
	"time"

	"solana-balance-api/pkg/metrics"

	"github.com/gin-gonic/gin"
)

// MetricsMiddleware creates a middleware that tracks request metrics
func MetricsMiddleware(metricsCollector *metrics.MetricsCollector) gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()

		// Record request start
		metricsCollector.RecordRequest()

		// Process request
		c.Next()

		// Calculate duration
		duration := time.Since(startTime)

		// Determine if request was successful (status code < 400)
		success := c.Writer.Status() < 400

		// Record request completion
		metricsCollector.RecordRequestComplete(duration, success)
	}
}
