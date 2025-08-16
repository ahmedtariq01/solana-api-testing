# Solana Balance API

A high-performance REST API built in Go that fetches Solana wallet balances using the Helius RPC endpoint. The API includes authentication, rate limiting, caching, and concurrent request handling.

## Project Structure

```
solana-balance-api/
├── cmd/
│   └── server/           # Application entrypoint
│       └── main.go
├── internal/
│   ├── config/           # Configuration management
│   ├── handlers/         # HTTP request handlers
│   ├── middleware/       # HTTP middleware (auth, rate limiting, etc.)
│   ├── models/           # Data models and structs
│   └── services/         # Business logic services
├── pkg/
│   ├── cache/            # Caching utilities
│   └── ratelimiter/      # Rate limiting utilities
├── .env.example          # Environment variables template
├── go.mod                # Go module definition
├── go.sum                # Go module checksums
└── README.md             # This file
```

## Dependencies

- **Gin**: HTTP web framework
- **MongoDB Driver**: Database connectivity
- **Solana-Go**: Solana blockchain integration
- **GoDotEnv**: Environment variable management

## Configuration

Copy `.env.example` to `.env` and adjust the values as needed:

```bash
cp .env.example .env
```

## Running the Application

```bash
go run cmd/server/main.go
```

## Features

- MongoDB-based API key authentication
- IP-based rate limiting (10 requests per minute)
- In-memory caching with 10-second TTL
- Concurrent request deduplication
- Helius Solana RPC integration
- High-performance HTTP server with Gin

## API Endpoints

- `POST /api/get-balance` - Fetch balance for one or multiple Solana wallets

## Development

This project follows Go best practices with a clean architecture:

- `cmd/` - Application entry points
- `internal/` - Private application code
- `pkg/` - Public library code that can be imported by other projects