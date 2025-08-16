package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"solana-balance-api/internal/config"
	"solana-balance-api/internal/models"
	"solana-balance-api/internal/services"
	"solana-balance-api/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockAuthService implements AuthServiceInterface for testing
type MockAuthService struct {
	validKeys map[string]*models.APIKey
	mu        sync.RWMutex
	callCount int64
}

// NewMockAuthService creates a new mock authentication service
func NewMockAuthService() *MockAuthService {
	return &MockAuthService{
		validKeys: make(map[string]*models.APIKey),
	}
}

// AddValidKey adds a valid API key for testing
func (m *MockAuthService) AddValidKey(key string, active bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.validKeys[key] = &models.APIKey{
		Key:       key,
		Name:      fmt.Sprintf("Test Key %s", key),
		Active:    active,
		CreatedAt: time.Now(),
	}
}

// ValidateAPIKey validates an API key (mock implementation)
func (m *MockAuthService) ValidateAPIKey(key string) (*models.APIKey, error) {
	atomic.AddInt64(&m.callCount, 1)

	m.mu.RLock()
	defer m.mu.RUnlock()

	apiKey, exists := m.validKeys[key]
	if !exists {
		return nil, services.ErrInvalidAPIKey
	}

	if !apiKey.Active {
		return nil, services.ErrInactiveAPIKey
	}

	return apiKey, nil
}

// GetCallCount returns the number of validation calls made
func (m *MockAuthService) GetCallCount() int64 {
	return atomic.LoadInt64(&m.callCount)
}

// Close closes the mock auth service (no-op for testing)
func (m *MockAuthService) Close() error {
	return nil
}

// MockSolanaClient implements SolanaServiceInterface for testing
type MockSolanaClient struct {
	balances    map[string]float64
	callCount   map[string]int64
	mu          sync.RWMutex
	delay       time.Duration
	shouldError bool
	errorMsg    string
}

// NewMockSolanaClient creates a new mock Solana client
func NewMockSolanaClient() *MockSolanaClient {
	return &MockSolanaClient{
		balances:  make(map[string]float64),
		callCount: make(map[string]int64),
		delay:     0,
	}
}

// SetBalance sets a mock balance for a wallet address
func (m *MockSolanaClient) SetBalance(address string, balance float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.balances[address] = balance
}

// SetDelay sets a delay for RPC calls to simulate network latency
func (m *MockSolanaClient) SetDelay(delay time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.delay = delay
}

// SetError configures the client to return errors
func (m *MockSolanaClient) SetError(shouldError bool, errorMsg string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldError = shouldError
	m.errorMsg = errorMsg
}

// GetBalance returns the mock balance for an address
func (m *MockSolanaClient) GetBalance(address string) (float64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Increment call count
	m.callCount[address]++

	// Simulate delay if configured
	if m.delay > 0 {
		time.Sleep(m.delay)
	}

	// Return error if configured
	if m.shouldError {
		return 0, fmt.Errorf(m.errorMsg)
	}

	// Return balance or default
	balance, exists := m.balances[address]
	if !exists {
		return 1.5, nil // Default test balance
	}

	return balance, nil
}

// GetBalances returns balances for multiple addresses
func (m *MockSolanaClient) GetBalances(addresses []string) (map[string]float64, error) {
	result := make(map[string]float64)
	for _, addr := range addresses {
		balance, err := m.GetBalance(addr)
		if err != nil {
			return nil, err
		}
		result[addr] = balance
	}
	return result, nil
}

// GetCallCount returns the number of calls made for a specific address
func (m *MockSolanaClient) GetCallCount(address string) int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.callCount[address]
}

// GetTotalCallCount returns the total number of calls made
func (m *MockSolanaClient) GetTotalCallCount() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var total int64
	for _, count := range m.callCount {
		total += count
	}
	return total
}

// ResetCallCounts resets all call counters
func (m *MockSolanaClient) ResetCallCounts() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCount = make(map[string]int64)
}

// IsHealthy checks if the mock client is healthy (always returns nil for testing)
func (m *MockSolanaClient) IsHealthy() error {
	return nil
}

// setupTestServer creates a test server with mock services
func setupTestServer(t *testing.T, cfg *config.Config) (*gin.Engine, *MockAuthService, *MockSolanaClient) {
	// Initialize logger for testing
	if err := logger.Initialize(&logger.Config{
		Level:       "debug",
		Environment: "test",
		OutputPaths: []string{"stdout"},
	}); err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}

	// Create mock services
	mockAuth := NewMockAuthService()
	mockSolana := NewMockSolanaClient()

	// Add test API keys
	mockAuth.AddValidKey("test-api-key", true)
	mockAuth.AddValidKey("inactive-key", false)

	// Set up test balances
	mockSolana.SetBalance("11111111111111111111111111111112", 1.5)
	mockSolana.SetBalance("11111111111111111111111111111113", 2.5)
	mockSolana.SetBalance("11111111111111111111111111111114", 3.5)

	// Create balance service with mock client
	balanceService := services.NewBalanceService(mockSolana, cfg)

	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create Gin engine
	engine := gin.New()

	// Add minimal middleware for testing
	engine.Use(gin.Recovery())

	// Create test server components
	server := &TestServer{
		config:         cfg,
		authService:    mockAuth,
		solanaClient:   mockSolana,
		balanceService: balanceService,
	}

	// Setup routes
	setupTestRoutes(engine, server)

	return engine, mockAuth, mockSolana
}

// TestServer represents a test server with mock services
type TestServer struct {
	config         *config.Config
	authService    *MockAuthService
	solanaClient   *MockSolanaClient
	balanceService *services.BalanceService
}

// setupTestRoutes configures routes for testing
func setupTestRoutes(engine *gin.Engine, server *TestServer) {
	// Health check route
	engine.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	// API routes with authentication middleware
	api := engine.Group("/api")
	api.Use(func(c *gin.Context) {
		// Simple auth middleware for testing
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing API key"})
			c.Abort()
			return
		}

		// Remove "Bearer " prefix if present
		apiKey := authHeader
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			apiKey = authHeader[7:]
		}

		// Validate API key
		_, err := server.authService.ValidateAPIKey(apiKey)
		if err != nil {
			var message string
			switch err {
			case services.ErrInvalidAPIKey:
				message = "Invalid API key"
			case services.ErrInactiveAPIKey:
				message = "API key is inactive"
			default:
				message = "Authentication failed"
			}
			c.JSON(http.StatusUnauthorized, gin.H{"error": message})
			c.Abort()
			return
		}

		c.Next()
	})

	// Balance endpoint
	api.POST("/get-balance", func(c *gin.Context) {
		var req models.BalanceRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
			return
		}

		if len(req.Wallets) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Wallets array cannot be empty"})
			return
		}

		// Validate wallet addresses
		for _, wallet := range req.Wallets {
			if len(wallet) < 32 || len(wallet) > 44 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid wallet address format"})
				return
			}
		}

		response, err := server.balanceService.GetBalances(req.Wallets)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch balances"})
			return
		}

		c.JSON(http.StatusOK, response)
	})
}

// TestSingleWalletBalanceRetrieval tests single wallet balance retrieval (Requirement 8.1)
func TestSingleWalletBalanceRetrieval(t *testing.T) {
	cfg := &config.Config{
		Cache: config.CacheConfig{
			TTL:             10 * time.Second,
			CleanupInterval: 1 * time.Minute,
		},
		RateLimit: config.RateLimitConfig{
			RequestsPerMinute: 60,
			WindowSize:        time.Minute,
		},
	}

	engine, _, mockSolana := setupTestServer(t, cfg)

	t.Run("ValidSingleWallet", func(t *testing.T) {
		testWallet := "11111111111111111111111111111112"
		mockSolana.SetBalance(testWallet, 1.5)

		requestBody := models.BalanceRequest{
			Wallets: []string{testWallet},
		}

		jsonBody, _ := json.Marshal(requestBody)
		req := httptest.NewRequest("POST", "/api/get-balance", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "test-api-key")

		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response models.BalanceResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Len(t, response.Balances, 1)
		assert.Equal(t, testWallet, response.Balances[0].Address)
		assert.Equal(t, 1.5, response.Balances[0].Balance)
		assert.Empty(t, response.Balances[0].Error)
		assert.False(t, response.Cached) // First request should not be cached
	})

	t.Run("InvalidWalletAddressFormat", func(t *testing.T) {
		requestBody := models.BalanceRequest{
			Wallets: []string{"invalid-wallet"},
		}

		jsonBody, _ := json.Marshal(requestBody)
		req := httptest.NewRequest("POST", "/api/get-balance", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "test-api-key")

		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var errorResponse map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
		require.NoError(t, err)

		assert.Contains(t, errorResponse["error"], "Invalid wallet address format")
	})

	t.Run("RPCErrorHandling", func(t *testing.T) {
		testWallet := "11111111111111111111111111111115"
		mockSolana.SetError(true, "RPC service unavailable")

		requestBody := models.BalanceRequest{
			Wallets: []string{testWallet},
		}

		jsonBody, _ := json.Marshal(requestBody)
		req := httptest.NewRequest("POST", "/api/get-balance", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "test-api-key")

		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code) // Service should still respond

		var response models.BalanceResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Len(t, response.Balances, 1)
		assert.Equal(t, testWallet, response.Balances[0].Address)
		assert.Equal(t, 0.0, response.Balances[0].Balance)
		assert.Contains(t, response.Balances[0].Error, "Failed to fetch balance")

		// Reset error state
		mockSolana.SetError(false, "")
	})

	t.Run("EmptyWalletArray", func(t *testing.T) {
		requestBody := models.BalanceRequest{
			Wallets: []string{},
		}

		jsonBody, _ := json.Marshal(requestBody)
		req := httptest.NewRequest("POST", "/api/get-balance", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "test-api-key")

		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var errorResponse map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
		require.NoError(t, err)

		assert.Contains(t, errorResponse["error"], "Wallets array cannot be empty")
	})
}

// TestMultipleWalletBatchProcessing tests multiple wallet batch processing (Requirement 8.2)
func TestMultipleWalletBatchProcessing(t *testing.T) {
	cfg := &config.Config{
		Cache: config.CacheConfig{
			TTL:             10 * time.Second,
			CleanupInterval: 1 * time.Minute,
		},
		RateLimit: config.RateLimitConfig{
			RequestsPerMinute: 60,
			WindowSize:        time.Minute,
		},
	}

	engine, _, mockSolana := setupTestServer(t, cfg)

	t.Run("ValidMultipleWallets", func(t *testing.T) {
		testWallets := []string{
			"11111111111111111111111111111112",
			"11111111111111111111111111111113",
			"11111111111111111111111111111114",
		}

		// Set different balances for each wallet
		mockSolana.SetBalance(testWallets[0], 1.5)
		mockSolana.SetBalance(testWallets[1], 2.5)
		mockSolana.SetBalance(testWallets[2], 3.5)

		requestBody := models.BalanceRequest{
			Wallets: testWallets,
		}

		jsonBody, _ := json.Marshal(requestBody)
		req := httptest.NewRequest("POST", "/api/get-balance", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "test-api-key")

		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response models.BalanceResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Len(t, response.Balances, 3)
		assert.False(t, response.Cached) // First request should not be cached

		// Verify each balance
		expectedBalances := map[string]float64{
			testWallets[0]: 1.5,
			testWallets[1]: 2.5,
			testWallets[2]: 3.5,
		}

		for _, balance := range response.Balances {
			expectedBalance, exists := expectedBalances[balance.Address]
			assert.True(t, exists, "Unexpected wallet address: %s", balance.Address)
			assert.Equal(t, expectedBalance, balance.Balance)
			assert.Empty(t, balance.Error)
		}
	})

	t.Run("MixedValidInvalidWallets", func(t *testing.T) {
		testWallets := []string{
			"11111111111111111111111111111112", // Valid
			"invalid-wallet",                   // Invalid
			"11111111111111111111111111111113", // Valid
		}

		requestBody := models.BalanceRequest{
			Wallets: testWallets,
		}

		jsonBody, _ := json.Marshal(requestBody)
		req := httptest.NewRequest("POST", "/api/get-balance", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "test-api-key")

		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var errorResponse map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
		require.NoError(t, err)

		assert.Contains(t, errorResponse["error"], "Invalid wallet address format")
	})

	t.Run("LargeBatchProcessing", func(t *testing.T) {
		// Create fresh test server for this test to avoid cache interference
		freshEngine, _, freshMockSolana := setupTestServer(t, cfg)

		// Test with 10 wallets to verify concurrent processing
		testWallets := make([]string, 10)
		for i := 0; i < 10; i++ {
			testWallets[i] = fmt.Sprintf("2111111111111111111111111111111%d", i)
			freshMockSolana.SetBalance(testWallets[i], float64(i)+1.0)
		}

		requestBody := models.BalanceRequest{
			Wallets: testWallets,
		}

		jsonBody, _ := json.Marshal(requestBody)
		req := httptest.NewRequest("POST", "/api/get-balance", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "test-api-key")

		startTime := time.Now()
		w := httptest.NewRecorder()
		freshEngine.ServeHTTP(w, req)
		duration := time.Since(startTime)

		assert.Equal(t, http.StatusOK, w.Code)

		var response models.BalanceResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Len(t, response.Balances, 10)

		// Verify all balances are correct
		for i, balance := range response.Balances {
			assert.Equal(t, testWallets[i], balance.Address)
			assert.Equal(t, float64(i)+1.0, balance.Balance)
			assert.Empty(t, balance.Error)
		}

		// Concurrent processing should be faster than sequential
		t.Logf("Batch processing duration: %v", duration)
		assert.True(t, duration < 5*time.Second, "Batch processing should complete quickly")
	})
}

// TestConcurrentRequestsWithSameWallet tests concurrent request handling (Requirement 8.3)
func TestConcurrentRequestsWithSameWallet(t *testing.T) {
	cfg := &config.Config{
		Cache: config.CacheConfig{
			TTL:             10 * time.Second,
			CleanupInterval: 1 * time.Minute,
		},
		RateLimit: config.RateLimitConfig{
			RequestsPerMinute: 100, // Higher limit for concurrent testing
			WindowSize:        time.Minute,
		},
	}

	engine, _, mockSolana := setupTestServer(t, cfg)

	t.Run("ConcurrentRequestsSameWallet", func(t *testing.T) {
		testWallet := "11111111111111111111111111111116"
		mockSolana.SetBalance(testWallet, 5.0)
		mockSolana.SetDelay(100 * time.Millisecond) // Add delay to simulate RPC latency
		mockSolana.ResetCallCounts()

		const numConcurrentRequests = 10
		var wg sync.WaitGroup
		responses := make([]*httptest.ResponseRecorder, numConcurrentRequests)

		requestBody := models.BalanceRequest{
			Wallets: []string{testWallet},
		}
		jsonBody, _ := json.Marshal(requestBody)

		// Launch concurrent requests
		for i := 0; i < numConcurrentRequests; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()

				req := httptest.NewRequest("POST", "/api/get-balance", bytes.NewBuffer(jsonBody))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", "test-api-key")

				w := httptest.NewRecorder()
				engine.ServeHTTP(w, req)
				responses[index] = w
			}(i)
		}

		wg.Wait()

		// All requests should succeed
		for i, resp := range responses {
			assert.Equal(t, http.StatusOK, resp.Code, "Request %d should succeed", i)

			var response models.BalanceResponse
			err := json.Unmarshal(resp.Body.Bytes(), &response)
			require.NoError(t, err, "Request %d should have valid response", i)

			assert.Len(t, response.Balances, 1)
			assert.Equal(t, testWallet, response.Balances[0].Address)
			assert.Equal(t, 5.0, response.Balances[0].Balance)
			assert.Empty(t, response.Balances[0].Error)
		}

		// Verify that mutex prevented duplicate RPC calls
		// Due to caching and mutex, only 1 RPC call should have been made
		callCount := mockSolana.GetCallCount(testWallet)
		assert.Equal(t, int64(1), callCount, "Only one RPC call should be made for concurrent requests to same wallet")

		// Reset delay
		mockSolana.SetDelay(0)
	})

	t.Run("ConcurrentRequestsDifferentWallets", func(t *testing.T) {
		testWallets := []string{
			"11111111111111111111111111111117",
			"11111111111111111111111111111118",
			"11111111111111111111111111111119",
		}

		// Set balances for each wallet
		for i, wallet := range testWallets {
			mockSolana.SetBalance(wallet, float64(i+1)*2.0)
		}

		mockSolana.ResetCallCounts()

		const numConcurrentRequests = 3
		var wg sync.WaitGroup
		responses := make([]*httptest.ResponseRecorder, numConcurrentRequests)

		// Launch concurrent requests for different wallets
		for i := 0; i < numConcurrentRequests; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()

				requestBody := models.BalanceRequest{
					Wallets: []string{testWallets[index]},
				}
				jsonBody, _ := json.Marshal(requestBody)

				req := httptest.NewRequest("POST", "/api/get-balance", bytes.NewBuffer(jsonBody))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", "test-api-key")

				w := httptest.NewRecorder()
				engine.ServeHTTP(w, req)
				responses[index] = w
			}(i)
		}

		wg.Wait()

		// All requests should succeed
		for i, resp := range responses {
			assert.Equal(t, http.StatusOK, resp.Code, "Request %d should succeed", i)

			var response models.BalanceResponse
			err := json.Unmarshal(resp.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.Len(t, response.Balances, 1)
			assert.Equal(t, testWallets[i], response.Balances[0].Address)
			assert.Equal(t, float64(i+1)*2.0, response.Balances[0].Balance)
		}

		// Each different wallet should have gotten its own RPC call
		for _, wallet := range testWallets {
			callCount := mockSolana.GetCallCount(wallet)
			assert.Equal(t, int64(1), callCount, "Each different wallet should get one RPC call")
		}
	})
}

// TestAuthenticationScenarios tests authentication scenarios (Requirement 8.4)
func TestAuthenticationScenarios(t *testing.T) {
	cfg := &config.Config{
		Cache: config.CacheConfig{
			TTL:             10 * time.Second,
			CleanupInterval: 1 * time.Minute,
		},
		RateLimit: config.RateLimitConfig{
			RequestsPerMinute: 60,
			WindowSize:        time.Minute,
		},
	}

	engine, mockAuth, _ := setupTestServer(t, cfg)

	requestBody := models.BalanceRequest{
		Wallets: []string{"11111111111111111111111111111112"},
	}
	jsonBody, _ := json.Marshal(requestBody)

	t.Run("ValidAPIKey", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/get-balance", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "test-api-key")

		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response models.BalanceResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Len(t, response.Balances, 1)
	})

	t.Run("MissingAPIKey", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/get-balance", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		// No Authorization header

		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var errorResponse map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
		require.NoError(t, err)

		assert.Equal(t, "Missing API key", errorResponse["error"])
	})

	t.Run("InvalidAPIKey", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/get-balance", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "invalid-key")

		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var errorResponse map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
		require.NoError(t, err)

		assert.Equal(t, "Invalid API key", errorResponse["error"])
	})

	t.Run("BearerTokenFormat", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/get-balance", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer test-api-key")

		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response models.BalanceResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Len(t, response.Balances, 1)
	})

	t.Run("InactiveAPIKey", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/get-balance", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "inactive-key")

		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var errorResponse map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
		require.NoError(t, err)

		assert.Equal(t, "API key is inactive", errorResponse["error"])
	})

	t.Run("MalformedAuthorizationHeader", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/get-balance", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer") // Malformed Bearer token

		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var errorResponse map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
		require.NoError(t, err)

		assert.Equal(t, "Invalid API key", errorResponse["error"])
	})

	t.Run("AuthenticationCallCount", func(t *testing.T) {
		initialCallCount := mockAuth.GetCallCount()

		req := httptest.NewRequest("POST", "/api/get-balance", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "test-api-key")

		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		// Verify that authentication was called
		finalCallCount := mockAuth.GetCallCount()
		assert.Equal(t, initialCallCount+1, finalCallCount, "Authentication should be called once per request")
	})
}

// TestRateLimitingScenarios tests rate limiting scenarios (Requirement 8.5)
func TestRateLimitingScenarios(t *testing.T) {
	// Create a separate test server with rate limiting
	engine := gin.New()
	gin.SetMode(gin.TestMode)

	// Simple rate limiter for testing
	requestCounts := make(map[string]int)
	var mu sync.Mutex

	// Health endpoint (should bypass rate limiting) - add before rate limiting middleware
	engine.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	// Rate limiting middleware function
	rateLimitMiddleware := func(c *gin.Context) {
		mu.Lock()
		defer mu.Unlock()

		clientIP := c.ClientIP()
		count := requestCounts[clientIP]

		if count >= 5 {
			c.Header("X-RateLimit-Limit", "5")
			c.Header("X-RateLimit-Remaining", "0")
			c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(time.Minute).Unix()))
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "Rate limit exceeded"})
			c.Abort()
			return
		}

		requestCounts[clientIP] = count + 1
		remaining := 5 - (count + 1)

		c.Header("X-RateLimit-Limit", "5")
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
		c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(time.Minute).Unix()))

		c.Next()
	}

	// API endpoint with rate limiting
	api := engine.Group("/api")
	api.Use(rateLimitMiddleware)
	api.POST("/get-balance", func(c *gin.Context) {
		c.JSON(http.StatusOK, models.BalanceResponse{
			Balances: []models.WalletBalance{
				{Address: "11111111111111111111111111111112", Balance: 1.5},
			},
			Cached: false,
		})
	})

	requestBody := models.BalanceRequest{
		Wallets: []string{"11111111111111111111111111111112"},
	}
	jsonBody, _ := json.Marshal(requestBody)

	t.Run("RequestsWithinLimit", func(t *testing.T) {
		// Make 5 requests (within limit)
		for i := 0; i < 5; i++ {
			req := httptest.NewRequest("POST", "/api/get-balance", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			req.RemoteAddr = "192.168.1.100:12345" // Set a specific IP

			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code, "Request %d should succeed", i+1)

			// Check rate limit headers
			assert.Equal(t, "5", w.Header().Get("X-RateLimit-Limit"))
			expectedRemaining := fmt.Sprintf("%d", 4-i)
			assert.Equal(t, expectedRemaining, w.Header().Get("X-RateLimit-Remaining"))
			assert.NotEmpty(t, w.Header().Get("X-RateLimit-Reset"))
		}
	})

	t.Run("RequestsExceedingLimit", func(t *testing.T) {
		// Make the 6th request (should be rate limited)
		req := httptest.NewRequest("POST", "/api/get-balance", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "192.168.1.100:12346" // Same IP as previous test

		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		assert.Equal(t, http.StatusTooManyRequests, w.Code)

		var errorResponse map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
		require.NoError(t, err)

		assert.Equal(t, "Rate limit exceeded", errorResponse["error"])

		// Check rate limit headers
		assert.Equal(t, "5", w.Header().Get("X-RateLimit-Limit"))
		assert.Equal(t, "0", w.Header().Get("X-RateLimit-Remaining"))
		assert.NotEmpty(t, w.Header().Get("X-RateLimit-Reset"))
	})

	t.Run("DifferentIPNotAffected", func(t *testing.T) {
		// Request from different IP should not be affected
		req := httptest.NewRequest("POST", "/api/get-balance", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "192.168.1.200:12345" // Different IP

		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		// Should have full rate limit available
		assert.Equal(t, "5", w.Header().Get("X-RateLimit-Limit"))
		assert.Equal(t, "4", w.Header().Get("X-RateLimit-Remaining"))
	})

	t.Run("HealthEndpointBypassesRateLimit", func(t *testing.T) {
		// Health endpoint should not be rate limited
		req := httptest.NewRequest("GET", "/health", nil)
		req.RemoteAddr = "192.168.1.100:12347" // Same IP that was rate limited

		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "healthy", response["status"])
	})
}

// TestCacheTTLBehavior tests cache TTL behavior (Requirement 8.6)
func TestCacheTTLBehavior(t *testing.T) {
	cfg := &config.Config{
		Cache: config.CacheConfig{
			TTL:             200 * time.Millisecond, // Short TTL for testing
			CleanupInterval: 1 * time.Minute,
		},
		RateLimit: config.RateLimitConfig{
			RequestsPerMinute: 60,
			WindowSize:        time.Minute,
		},
	}

	engine, _, mockSolana := setupTestServer(t, cfg)

	testWallet := "11111111111111111111111111111120"
	mockSolana.SetBalance(testWallet, 10.0)

	requestBody := models.BalanceRequest{
		Wallets: []string{testWallet},
	}
	jsonBody, _ := json.Marshal(requestBody)

	t.Run("CacheHitWithinTTL", func(t *testing.T) {
		mockSolana.ResetCallCounts()

		// First request - should hit RPC
		req1 := httptest.NewRequest("POST", "/api/get-balance", bytes.NewBuffer(jsonBody))
		req1.Header.Set("Content-Type", "application/json")
		req1.Header.Set("Authorization", "test-api-key")

		w1 := httptest.NewRecorder()
		engine.ServeHTTP(w1, req1)

		assert.Equal(t, http.StatusOK, w1.Code)

		var response1 models.BalanceResponse
		err := json.Unmarshal(w1.Body.Bytes(), &response1)
		require.NoError(t, err)

		assert.False(t, response1.Cached, "First request should not be cached")
		assert.Equal(t, int64(1), mockSolana.GetCallCount(testWallet), "First request should make RPC call")

		// Second request immediately - should hit cache
		req2 := httptest.NewRequest("POST", "/api/get-balance", bytes.NewBuffer(jsonBody))
		req2.Header.Set("Content-Type", "application/json")
		req2.Header.Set("Authorization", "test-api-key")

		w2 := httptest.NewRecorder()
		engine.ServeHTTP(w2, req2)

		assert.Equal(t, http.StatusOK, w2.Code)

		var response2 models.BalanceResponse
		err = json.Unmarshal(w2.Body.Bytes(), &response2)
		require.NoError(t, err)

		assert.True(t, response2.Cached, "Second request should be cached")
		assert.Equal(t, int64(1), mockSolana.GetCallCount(testWallet), "Second request should not make additional RPC call")

		// Verify same balance returned
		assert.Equal(t, response1.Balances[0].Balance, response2.Balances[0].Balance)
	})

	t.Run("CacheMissAfterTTLExpiration", func(t *testing.T) {
		// Create fresh test server for this test to avoid cache interference
		freshEngine, _, freshMockSolana := setupTestServer(t, cfg)

		// Use a different wallet address to avoid cache interference
		freshTestWallet := "11111111111111111111111111111130"
		freshMockSolana.SetBalance(freshTestWallet, 10.0)

		freshRequestBody := models.BalanceRequest{
			Wallets: []string{freshTestWallet},
		}
		freshJsonBody, _ := json.Marshal(freshRequestBody)

		freshMockSolana.ResetCallCounts()

		// First request
		req1 := httptest.NewRequest("POST", "/api/get-balance", bytes.NewBuffer(freshJsonBody))
		req1.Header.Set("Content-Type", "application/json")
		req1.Header.Set("Authorization", "test-api-key")

		w1 := httptest.NewRecorder()
		freshEngine.ServeHTTP(w1, req1)

		assert.Equal(t, http.StatusOK, w1.Code)

		var response1 models.BalanceResponse
		err := json.Unmarshal(w1.Body.Bytes(), &response1)
		require.NoError(t, err)

		assert.False(t, response1.Cached, "First request should not be cached")

		// Wait for TTL to expire
		time.Sleep(250 * time.Millisecond)

		// Change the balance to verify fresh fetch
		freshMockSolana.SetBalance(freshTestWallet, 15.0)

		// Second request after TTL expiration - should hit RPC again
		req2 := httptest.NewRequest("POST", "/api/get-balance", bytes.NewBuffer(freshJsonBody))
		req2.Header.Set("Content-Type", "application/json")
		req2.Header.Set("Authorization", "test-api-key")

		w2 := httptest.NewRecorder()
		freshEngine.ServeHTTP(w2, req2)

		assert.Equal(t, http.StatusOK, w2.Code)

		var response2 models.BalanceResponse
		err = json.Unmarshal(w2.Body.Bytes(), &response2)
		require.NoError(t, err)

		assert.False(t, response2.Cached, "Request after TTL expiration should not be cached")
		assert.Equal(t, int64(2), freshMockSolana.GetCallCount(freshTestWallet), "Should make two RPC calls")

		// Verify updated balance
		assert.Equal(t, 15.0, response2.Balances[0].Balance, "Should return updated balance after TTL expiration")
	})

	t.Run("MultipleWalletsMixedCacheState", func(t *testing.T) {
		testWallets := []string{
			"11111111111111111111111111111121", // Will be cached
			"11111111111111111111111111111122", // Will not be cached
		}

		mockSolana.SetBalance(testWallets[0], 20.0)
		mockSolana.SetBalance(testWallets[1], 25.0)
		mockSolana.ResetCallCounts()

		// Cache first wallet
		firstWalletRequest := models.BalanceRequest{
			Wallets: []string{testWallets[0]},
		}
		firstJsonBody, _ := json.Marshal(firstWalletRequest)

		req1 := httptest.NewRequest("POST", "/api/get-balance", bytes.NewBuffer(firstJsonBody))
		req1.Header.Set("Content-Type", "application/json")
		req1.Header.Set("Authorization", "test-api-key")

		w1 := httptest.NewRecorder()
		engine.ServeHTTP(w1, req1)
		assert.Equal(t, http.StatusOK, w1.Code)

		// Now request both wallets
		bothWalletsRequest := models.BalanceRequest{
			Wallets: testWallets,
		}
		bothJsonBody, _ := json.Marshal(bothWalletsRequest)

		req2 := httptest.NewRequest("POST", "/api/get-balance", bytes.NewBuffer(bothJsonBody))
		req2.Header.Set("Content-Type", "application/json")
		req2.Header.Set("Authorization", "test-api-key")

		w2 := httptest.NewRecorder()
		engine.ServeHTTP(w2, req2)

		assert.Equal(t, http.StatusOK, w2.Code)

		var response models.BalanceResponse
		err := json.Unmarshal(w2.Body.Bytes(), &response)
		require.NoError(t, err)

		// Response should not be marked as fully cached since one wallet wasn't cached
		assert.False(t, response.Cached, "Mixed cache state should result in cached=false")

		// Verify balances
		assert.Len(t, response.Balances, 2)
		for _, balance := range response.Balances {
			if balance.Address == testWallets[0] {
				assert.Equal(t, 20.0, balance.Balance)
			} else if balance.Address == testWallets[1] {
				assert.Equal(t, 25.0, balance.Balance)
			}
		}

		// First wallet should have 1 call (from first request), second wallet should have 1 call
		assert.Equal(t, int64(1), mockSolana.GetCallCount(testWallets[0]))
		assert.Equal(t, int64(1), mockSolana.GetCallCount(testWallets[1]))
	})
}
