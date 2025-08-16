# Solana Balance API Server

A high-performance REST API server for fetching Solana wallet balances with authentication, rate limiting, caching, and concurrency control.

## Features

- **Authentication**: MongoDB-based API key validation
- **Rate Limiting**: IP-based limiting (10 requests per minute by default)
- **Caching**: In-memory caching with 10-second TTL
- **Concurrency Control**: Request deduplication using mutexes
- **Graceful Shutdown**: Proper cleanup of resources
- **Health Monitoring**: Health check and status endpoints
- **Structured Logging**: Request logging with correlation
- **CORS Support**: Cross-origin resource sharing

## Quick Start

### Prerequisites

1. **Go 1.21+** installed
2. **MongoDB** running and accessible
3. **API Keys** configured in MongoDB

### Running the Server

```bash
# Build the server
go build -o server ./cmd/server

# Run with default configuration
./server

# Or run directly with Go
go run ./cmd/server
```

### Environment Configuration

The server can be configured using environment variables:

```bash
# Server Configuration
export SERVER_PORT=8080
export SERVER_HOST=0.0.0.0
export SERVER_READ_TIMEOUT=10s
export SERVER_WRITE_TIMEOUT=10s
export SERVER_IDLE_TIMEOUT=60s

# MongoDB Configuration
export MONGODB_URI=mongodb://localhost:27017
export MONGODB_DATABASE=solana_api
export MONGODB_APIKEY_COLLECTION=api_keys
export MONGODB_CONNECT_TIMEOUT=10s
export MONGODB_MAX_POOL_SIZE=100

# Solana RPC Configuration
export SOLANA_RPC_ENDPOINT=https://your-helius-endpoint
export SOLANA_RPC_TIMEOUT=30s

# Cache Configuration
export CACHE_TTL=10s
export CACHE_CLEANUP_INTERVAL=60s
export CACHE_MAX_SIZE=10000

# Rate Limiting Configuration
export RATE_LIMIT_REQUESTS_PER_MINUTE=10
export RATE_LIMIT_WINDOW_SIZE=1m
export RATE_LIMIT_CLEANUP_INTERVAL=5m

# Gin Mode (development/release)
export GIN_MODE=release
```

## API Endpoints

### Health Check

```http
GET /health
```

**Response:**
```json
{
  "status": "healthy",
  "service": "solana-balance-api"
}
```

### Server Status

```http
GET /status
```

**Response:**
```json
{
  "service": "solana-balance-api",
  "status": "running",
  "rpc_healthy": true,
  "uptime": "1h30m45s",
  "version": "1.0.0"
}
```

### Metrics

```http
GET /metrics
```

**Response:**
```json
{
  "service": "solana-balance-api",
  "cache": {
    "cache_size": 150,
    "mutex_count": 5,
    "cache_ttl_ms": 10000
  },
  "uptime": "1h30m45s"
}
```

### Get Balances

```http
POST /api/get-balance
Authorization: your-api-key
Content-Type: application/json
```

**Request Body:**
```json
{
  "wallets": [
    "11111111111111111111111111111112",
    "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA"
  ]
}
```

**Response:**
```json
{
  "balances": [
    {
      "address": "11111111111111111111111111111112",
      "balance": 0.0
    },
    {
      "address": "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA",
      "balance": 1.5
    }
  ],
  "cached": false
}
```

**Rate Limit Headers:**
```
X-RateLimit-Limit: 10
X-RateLimit-Remaining: 9
X-RateLimit-Reset: 1640995200
```

## Error Responses

### Authentication Errors (401)

```json
{
  "error": {
    "code": "INVALID_API_KEY",
    "message": "Invalid API key",
    "details": "API key not found in database"
  },
  "timestamp": "2023-12-01T10:30:00Z"
}
```

### Rate Limiting Errors (429)

```json
{
  "error": {
    "code": "RATE_LIMIT_EXCEEDED",
    "message": "Too many requests. Rate limit exceeded.",
    "details": "Maximum 10 requests per minute allowed."
  },
  "timestamp": "2023-12-01T10:30:00Z"
}
```

### Validation Errors (400)

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid request format",
    "details": "wallets field is required"
  },
  "timestamp": "2023-12-01T10:30:00Z"
}
```

## MongoDB Setup

### Create API Keys Collection

```javascript
// Connect to MongoDB
use solana_api

// Create API keys collection with index
db.api_keys.createIndex({ "key": 1 }, { unique: true })

// Insert a test API key
db.api_keys.insertOne({
  key: "test-api-key-12345",
  name: "Test API Key",
  active: true,
  created_at: new Date(),
  last_used: null
})
```

### API Key Document Structure

```json
{
  "_id": "ObjectId",
  "key": "unique-api-key-string",
  "name": "Human readable name",
  "active": true,
  "created_at": "2023-12-01T10:00:00Z",
  "last_used": "2023-12-01T10:30:00Z"
}
```

## Docker Deployment

### Dockerfile

```dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o server ./cmd/server

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/server .

EXPOSE 8080
CMD ["./server"]
```

### Docker Compose

```yaml
version: '3.8'

services:
  solana-api:
    build: .
    ports:
      - "8080:8080"
    environment:
      - MONGODB_URI=mongodb://mongo:27017
      - SOLANA_RPC_ENDPOINT=https://your-helius-endpoint
    depends_on:
      - mongo

  mongo:
    image: mongo:7
    ports:
      - "27017:27017"
    volumes:
      - mongo_data:/data/db

volumes:
  mongo_data:
```

## Performance Considerations

### Caching Strategy

- **TTL**: 10 seconds (configurable)
- **Storage**: In-memory with cleanup
- **Concurrency**: Thread-safe operations
- **Memory**: LRU eviction when max size reached

### Rate Limiting

- **Algorithm**: Sliding window
- **Granularity**: Per IP address
- **Storage**: In-memory with periodic cleanup
- **Headers**: Standard rate limit headers included

### Connection Pooling

- **MongoDB**: Configurable pool size (default: 100)
- **HTTP Client**: Keep-alive connections
- **Timeouts**: Configurable read/write/idle timeouts

## Monitoring and Logging

### Log Format

```
[2023-12-01 10:30:00] POST /api/get-balance 200 45.2ms 192.168.1.100
```

### Metrics Available

- Cache hit/miss ratios
- Request counts and latency
- Active mutex count
- Memory usage
- RPC health status

### Health Checks

The server provides multiple health check endpoints:

- `/health` - Basic health status
- `/status` - Detailed status with RPC health
- `/metrics` - Performance metrics

## Testing

### Unit Tests

```bash
# Run all tests
go test ./...

# Run server tests specifically
go test ./cmd/server -v

# Run with coverage
go test ./cmd/server -cover
```

### Integration Tests

```bash
# Run integration tests
go test ./cmd/server -run TestServerIntegration -v
```

### Load Testing

```bash
# Example with curl
for i in {1..20}; do
  curl -X POST http://localhost:8080/api/get-balance \
    -H "Authorization: test-api-key-12345" \
    -H "Content-Type: application/json" \
    -d '{"wallets":["11111111111111111111111111111112"]}' &
done
wait
```

## Troubleshooting

### Common Issues

1. **MongoDB Connection Failed**
   - Check MongoDB is running
   - Verify connection string
   - Check network connectivity

2. **Rate Limit Exceeded**
   - Check request frequency
   - Verify IP address handling
   - Adjust rate limit configuration

3. **RPC Timeout**
   - Check Helius endpoint availability
   - Verify API key in RPC URL
   - Increase timeout configuration

4. **High Memory Usage**
   - Monitor cache size
   - Check for memory leaks in mutexes
   - Adjust cleanup intervals

### Debug Mode

```bash
# Enable debug logging
export GIN_MODE=debug
go run ./cmd/server
```

### Graceful Shutdown

The server handles SIGINT and SIGTERM signals for graceful shutdown:

1. Stops accepting new requests
2. Completes in-flight requests
3. Closes database connections
4. Cleans up background routines
5. Exits cleanly

## Security Best Practices

1. **API Keys**: Use strong, unique keys
2. **Rate Limiting**: Adjust limits based on usage patterns
3. **CORS**: Configure allowed origins appropriately
4. **Timeouts**: Set reasonable timeout values
5. **Logging**: Avoid logging sensitive information
6. **TLS**: Use HTTPS in production (configure reverse proxy)

## Contributing

1. Follow Go coding standards
2. Add tests for new features
3. Update documentation
4. Ensure graceful error handling
5. Maintain backward compatibility