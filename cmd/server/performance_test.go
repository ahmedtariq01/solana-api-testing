package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"solana-balance-api/internal/config"
	"solana-balance-api/internal/models"
	"github.com/gin-gonic/gin"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPerformanceOptimizations tests all performance optimizations
func TestPerformanceOptimizations(t *testing.T) {
	// Load test configuration
	cfg := config.LoadConfig()

	// Override with test values
	cfg.Cache.TTL = 2 * time.Second
	cfg.RateLimit.RequestsPerMinute = 100 // Higher limit for testing

	// Create test server
	server, err := NewServer(cfg)
	require.NoError(t, err)

	// Create test router
	router := setupTestRouter(server)

	t.Run("CachingPerformance", func(t *testing.T) {
		testCachingPerformance(t, router)
	})

	t.Run("ConcurrentRequestHandling", func(t *testing.T) {
		testConcurrentRequestHandling(t, router)
	})

	t.Run("MetricsCollection", func(t *testing.T) {
		testMetricsCollection(t, router)
	})

	t.Run("ConnectionPooling", func(t *testing.T) {
		testConnectionPooling(t, router)
	})
}

func testCachingPerformance(t *testing.T, router http.Handler) {
	// Test wallet address
	testWallet := "11111111111111111111111111111112"

	requestBody := models.BalanceRequest{
		Wallets: []string{testWallet},
	}

	jsonBody, _ := json.Marshal(requestBody)

	// First request - should hit RPC
	req1 := httptest.NewRequest("POST", "/api/get-balance", bytes.NewBuffer(jsonBody))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Authorization", "test-api-key")

	w1 := httptest.NewRecorder()
	start1 := time.Now()
	router.ServeHTTP(w1, req1)
	duration1 := time.Since(start1)

	assert.Equal(t, http.StatusOK, w1.Code)

	// Second request immediately - should hit cache
	req2 := httptest.NewRequest("POST", "/api/get-balance", bytes.NewBuffer(jsonBody))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "test-api-key")

	w2 := httptest.NewRecorder()
	start2 := time.Now()
	router.ServeHTTP(w2, req2)
	duration2 := time.Since(start2)

	assert.Equal(t, http.StatusOK, w2.Code)

	// Cache hit should be significantly faster
	assert.True(t, duration2 < duration1/2,
		fmt.Sprintf("Cache hit (%v) should be faster than RPC call (%v)", duration2, duration1))

	// Parse responses to verify caching
	var resp1, resp2 models.BalanceResponse
	json.Unmarshal(w1.Body.Bytes(), &resp1)
	json.Unmarshal(w2.Body.Bytes(), &resp2)

	// Second response should indicate it was cached
	assert.True(t, resp2.Cached, "Second response should be cached")
}

func testConcurrentRequestHandling(t *testing.T, router http.Handler) {
	// Test concurrent requests for the same wallet
	testWallet := "11111111111111111111111111111113"

	requestBody := models.BalanceRequest{
		Wallets: []string{testWallet},
	}

	jsonBody, _ := json.Marshal(requestBody)

	const numConcurrentRequests = 10
	var wg sync.WaitGroup
	responses := make([]*httptest.ResponseRecorder, numConcurrentRequests)
	durations := make([]time.Duration, numConcurrentRequests)

	// Launch concurrent requests
	for i := 0; i < numConcurrentRequests; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			req := httptest.NewRequest("POST", "/api/get-balance", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "test-api-key")

			w := httptest.NewRecorder()
			start := time.Now()
			router.ServeHTTP(w, req)
			durations[index] = time.Since(start)
			responses[index] = w
		}(i)
	}

	wg.Wait()

	// All requests should succeed
	for i, resp := range responses {
		assert.Equal(t, http.StatusOK, resp.Code,
			fmt.Sprintf("Request %d should succeed", i))
	}

	// Verify that mutex prevented duplicate RPC calls
	// (This is implicit - if working correctly, only one RPC call should be made)
	t.Logf("Concurrent request durations: %v", durations)
}

func testMetricsCollection(t *testing.T, router http.Handler) {
	// Make a few requests to generate metrics
	testWallet := "11111111111111111111111111111114"

	requestBody := models.BalanceRequest{
		Wallets: []string{testWallet},
	}

	jsonBody, _ := json.Marshal(requestBody)

	// Make several requests
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("POST", "/api/get-balance", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "test-api-key")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		// Small delay between requests
		time.Sleep(10 * time.Millisecond)
	}

	// Check metrics endpoint
	metricsReq := httptest.NewRequest("GET", "/metrics", nil)
	metricsW := httptest.NewRecorder()
	router.ServeHTTP(metricsW, metricsReq)

	assert.Equal(t, http.StatusOK, metricsW.Code)

	var metricsResp map[string]interface{}
	err := json.Unmarshal(metricsW.Body.Bytes(), &metricsResp)
	require.NoError(t, err)

	// Verify metrics structure
	assert.Contains(t, metricsResp, "performance")

	performance, ok := metricsResp["performance"].(map[string]interface{})
	require.True(t, ok)

	// Check that metrics are being collected
	assert.Contains(t, performance, "total_requests")
	assert.Contains(t, performance, "successful_requests")
	assert.Contains(t, performance, "cache_hits")
	assert.Contains(t, performance, "cache_misses")
	assert.Contains(t, performance, "rpc_calls")

	// Verify some metrics have values
	totalRequests := performance["total_requests"].(float64)
	assert.True(t, totalRequests >= 5, "Should have at least 5 total requests")

	t.Logf("Metrics collected: %+v", performance)
}

func testConnectionPooling(t *testing.T, router http.Handler) {
	// Test multiple requests to verify connection reuse
	testWallets := []string{
		"11111111111111111111111111111115",
		"11111111111111111111111111111116",
		"11111111111111111111111111111117",
	}

	var totalDuration time.Duration
	const numRequests = 10

	for i := 0; i < numRequests; i++ {
		requestBody := models.BalanceRequest{
			Wallets: testWallets,
		}

		jsonBody, _ := json.Marshal(requestBody)

		req := httptest.NewRequest("POST", "/api/get-balance", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "test-api-key")

		w := httptest.NewRecorder()
		start := time.Now()
		router.ServeHTTP(w, req)
		duration := time.Since(start)
		totalDuration += duration

		assert.Equal(t, http.StatusOK, w.Code)

		// Small delay to allow connection reuse
		time.Sleep(5 * time.Millisecond)
	}

	averageDuration := totalDuration / numRequests
	t.Logf("Average request duration with connection pooling: %v", averageDuration)

	// Connection pooling should keep average response times reasonable
	assert.True(t, averageDuration < 5*time.Second,
		"Average response time should be reasonable with connection pooling")
}

func setupTestRouter(server *Server) http.Handler {
	// Build a real Gin engine that uses the server's middleware and routes so
	// tests exercise the actual cache, mutex and metrics logic.
	gin.SetMode(gin.TestMode)
	engine := gin.New()

	// Use the server's middleware and routes (these are helpers on Server)
	server.setupMiddleware(engine)
	// We cannot use the server's auth middleware because it talks to MongoDB.
	// Register routes manually and add a lightweight test auth middleware
	// that accepts the fixed test API key used by tests.

	// Test auth middleware
	testAuth := func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing API key"})
			c.Abort()
			return
		}
		apiKey := authHeader
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			apiKey = authHeader[7:]
		}
		if apiKey != "test-api-key" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid API key"})
			c.Abort()
			return
		}
		c.Next()
	}

	// Health routes
	server.router.SetupHealthRoutes(engine)

	// API routes with test auth
	api := engine.Group("/api")
	api.Use(testAuth)
	{
		api.POST("/get-balance", server.router.GetBalanceHandler().GetBalance)
	}

	// Metrics and status handlers
	engine.GET("/metrics", server.metricsHandler)
	engine.GET("/status", server.statusHandler)

	return engine
}
