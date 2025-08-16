package ratelimiter

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// Middleware creates a Gin middleware for rate limiting
func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get client IP
		clientIP := c.ClientIP()

		// Check if request is allowed
		if !rl.IsAllowed(clientIP) {
			// Get current request info for headers
			_, resetTime := rl.GetRequestInfo(clientIP)

			// Set rate limit headers
			c.Header("X-RateLimit-Limit", strconv.Itoa(rl.limit))
			c.Header("X-RateLimit-Remaining", "0")
			c.Header("X-RateLimit-Reset", strconv.FormatInt(resetTime.Unix(), 10))
			c.Header("Retry-After", strconv.Itoa(int(time.Until(resetTime).Seconds())))

			// Return 429 Too Many Requests
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": gin.H{
					"code":    "RATE_LIMIT_EXCEEDED",
					"message": "Too many requests. Rate limit exceeded.",
					"details": "Maximum " + strconv.Itoa(rl.limit) + " requests per minute allowed.",
				},
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			})
			c.Abort()
			return
		}

		// Get current request info for headers
		count, resetTime := rl.GetRequestInfo(clientIP)
		remaining := rl.limit - count
		if remaining < 0 {
			remaining = 0
		}

		// Set rate limit headers for successful requests
		c.Header("X-RateLimit-Limit", strconv.Itoa(rl.limit))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(resetTime.Unix(), 10))

		// Continue to next handler
		c.Next()
	}
}
