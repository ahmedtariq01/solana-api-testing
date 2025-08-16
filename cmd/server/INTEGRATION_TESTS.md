# Comprehensive Integration Tests

This document describes the comprehensive integration tests implemented for the Solana Balance API, covering all requirements specified in task 11.

## Test Coverage Overview

The integration tests are organized into 6 main test suites that cover all the requirements from the specification:

### 1. Single Wallet Balance Retrieval (Requirement 8.1)

**Test Function:** `TestSingleWalletBalanceRetrieval`

**Coverage:**
- Valid single wallet request with proper response format
- Invalid wallet address format validation
- RPC error handling and error response format
- Proper balance retrieval and caching behavior

**Key Scenarios:**
- Successful balance retrieval for a valid wallet address
- Validation of Solana address format (base58, 32-44 characters)
- Error handling when RPC service fails
- Proper error response structure with error details

### 2. Multiple Wallet Batch Processing (Requirement 8.2)

**Test Function:** `TestMultipleWalletBatchProcessing`

**Coverage:**
- Valid multiple wallet requests with concurrent processing
- Mixed valid and invalid wallet addresses handling
- Empty wallet array validation
- Large batch processing capabilities

**Key Scenarios:**
- Successful processing of multiple wallets in a single request
- Proper validation that rejects requests with any invalid addresses
- Rejection of empty wallet arrays
- Performance testing with larger batches (10 wallets)
- Concurrent processing of multiple wallets

### 3. Concurrent Requests with Same Wallet Address (Requirement 8.3)

**Test Function:** `TestConcurrentRequestsWithSameWallet`

**Coverage:**
- Request deduplication using mutex control
- Concurrent requests with different wallets
- Cache behavior under concurrent access
- Verification of single RPC call per wallet address

**Key Scenarios:**
- 10 concurrent requests for the same wallet result in only 1 RPC call
- Concurrent requests for different wallets each get their own RPC call
- Cache consistency under concurrent access
- Proper mutex behavior preventing race conditions

### 4. Authentication Scenarios (Requirement 8.4)

**Test Function:** `TestAuthenticationScenarios`

**Coverage:**
- Valid API key authentication
- Missing API key handling
- Invalid API key rejection
- Bearer token format support
- Inactive API key handling
- Malformed authorization header handling

**Key Scenarios:**
- Successful authentication with valid API key
- Proper 401 responses for missing API keys
- Proper 401 responses for invalid API keys
- Support for both direct API key and "Bearer <key>" formats
- Handling of inactive API keys
- Proper error messages and codes for different auth failures

### 5. Rate Limiting Scenarios (Requirement 8.5)

**Test Function:** `TestRateLimitingScenarios`

**Coverage:**
- Requests within rate limit succeed
- Requests exceeding rate limit are rejected
- Rate limit behavior verification
- Rate limit window reset functionality
- Health endpoints bypass rate limiting

**Key Scenarios:**
- Successful processing of requests within the configured limit
- Proper 429 responses when rate limit is exceeded
- Correct rate limit headers (X-RateLimit-Limit, X-RateLimit-Remaining, etc.)
- Rate limit window reset after time expiration
- Health endpoints are not subject to rate limiting

### 6. Cache TTL Behavior (Requirement 8.6)

**Test Function:** `TestCacheTTLBehavior`

**Coverage:**
- Cache hits within TTL period
- Cache misses after TTL expiration
- Cache behavior with multiple wallets
- Proper cache flag in responses

**Key Scenarios:**
- Requests within TTL period return cached results (cached=true)
- Requests after TTL expiration fetch fresh data (cached=false)
- Multiple wallet requests properly handle mixed cache states
- Cache TTL behavior is consistent across different wallet addresses

## Performance Testing

**Benchmark Function:** `BenchmarkBalanceAPI`

**Coverage:**
- Performance measurement of cached requests
- Concurrent request handling performance
- Memory usage and throughput testing

**Results:**
- Measures operations per second for cached balance requests
- Tests parallel request handling capabilities
- Validates performance under load

## Mock Services

The tests use comprehensive mock services that provide:

### MockAuthService
- Configurable valid/invalid API keys
- Support for active/inactive key states
- Proper error responses matching the real service

### MockSolanaClient
- Configurable balance responses
- Request counting for deduplication verification
- Configurable delays for concurrency testing
- Error simulation capabilities

## Test Configuration

The tests use different configurations for different scenarios:

- **Standard Tests:** 10 requests per minute rate limit
- **Concurrent Tests:** 100 requests per minute rate limit
- **Cache Tests:** 200ms TTL for faster testing
- **Benchmark Tests:** No rate limiting for performance measurement

## Running the Tests

```bash
# Run all integration tests
go test -v ./cmd/server -run "Test.*"

# Run specific test suites
go test -v ./cmd/server -run TestSingleWalletBalanceRetrieval
go test -v ./cmd/server -run TestMultipleWalletBatchProcessing
go test -v ./cmd/server -run TestConcurrentRequestsWithSameWallet
go test -v ./cmd/server -run TestAuthenticationScenarios
go test -v ./cmd/server -run TestRateLimitingScenarios
go test -v ./cmd/server -run TestCacheTTLBehavior

# Run performance benchmark
go test -v ./cmd/server -bench=BenchmarkBalanceAPI -benchtime=1s
```

## Test Assertions

Each test includes comprehensive assertions that verify:

- HTTP status codes match expected values
- Response body structure and content
- Error message format and content
- Rate limit headers presence and values
- Cache behavior flags
- Request deduplication effectiveness
- Timing behavior for TTL and rate limiting

## Requirements Mapping

| Requirement | Test Function | Coverage |
|-------------|---------------|----------|
| 8.1 | TestSingleWalletBalanceRetrieval | Single wallet balance retrieval |
| 8.2 | TestMultipleWalletBatchProcessing | Multiple wallet batch processing |
| 8.3 | TestConcurrentRequestsWithSameWallet | Concurrent request handling |
| 8.4 | TestAuthenticationScenarios | Authentication and authorization |
| 8.5 | TestRateLimitingScenarios | Rate limiting enforcement |
| 8.6 | TestCacheTTLBehavior | Cache TTL behavior |

All requirements from the specification are fully covered with comprehensive test scenarios that validate both positive and negative cases.