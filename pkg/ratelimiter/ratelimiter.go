package ratelimiter

import (
	"sync"
	"time"
)

// RequestCounter tracks request count and reset time for an IP
type RequestCounter struct {
	Count     int
	ResetTime time.Time
}

// RateLimiter implements IP-based rate limiting with in-memory tracking
type RateLimiter struct {
	requests map[string]*RequestCounter
	mutex    sync.RWMutex
	limit    int
	window   time.Duration
}

// New creates a new RateLimiter with specified limit and window
func New(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		requests: make(map[string]*RequestCounter),
		limit:    limit,
		window:   window,
	}
}

// IsAllowed checks if the IP address is allowed to make a request
// Returns true if allowed, false if rate limit exceeded
func (rl *RateLimiter) IsAllowed(ip string) bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()

	// Get or create request counter for this IP
	counter, exists := rl.requests[ip]
	if !exists {
		rl.requests[ip] = &RequestCounter{
			Count:     1,
			ResetTime: now.Add(rl.window),
		}
		return true
	}

	// Check if the window has expired
	if now.After(counter.ResetTime) {
		// Reset the counter for new window
		counter.Count = 1
		counter.ResetTime = now.Add(rl.window)
		return true
	}

	// Check if limit is exceeded
	if counter.Count >= rl.limit {
		return false
	}

	// Increment counter and allow request
	counter.Count++
	return true
}

// GetRequestInfo returns current request count and reset time for an IP
func (rl *RateLimiter) GetRequestInfo(ip string) (count int, resetTime time.Time) {
	rl.mutex.RLock()
	defer rl.mutex.RUnlock()

	counter, exists := rl.requests[ip]
	if !exists {
		return 0, time.Now().Add(rl.window)
	}

	// If window expired, return 0 count
	if time.Now().After(counter.ResetTime) {
		return 0, time.Now().Add(rl.window)
	}

	return counter.Count, counter.ResetTime
}

// Cleanup removes expired entries to prevent memory leaks
func (rl *RateLimiter) Cleanup() {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()
	for ip, counter := range rl.requests {
		if now.After(counter.ResetTime) {
			delete(rl.requests, ip)
		}
	}
}
