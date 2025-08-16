# Database Setup and Management

This directory contains scripts and utilities for managing the MongoDB database used by the Solana Balance API.

## Overview

The database setup includes:
- **Schema initialization** with required collections and indexes
- **Sample data seeding** for testing and development
- **Migration utilities** for schema changes
- **Health check functionality** for monitoring

## Quick Start

### Using the Database Setup Utility

The easiest way to set up the database is using the `dbsetup` utility:

```bash
# Full setup (recommended for first-time setup)
make db-setup

# Or run individual steps
make db-init    # Initialize schema
make db-seed    # Add test data
make db-health  # Check health
```

### Using Docker Compose

For development with Docker:

```bash
# Start MongoDB and the API
docker-compose up -d

# The MongoDB container will automatically:
# - Create the database and collections
# - Set up required indexes
# - Insert sample API keys
```

## Database Schema

### Collections

#### `api_keys`
Stores API keys for authentication.

**Fields:**
- `_id`: ObjectId (auto-generated)
- `key`: String (unique, indexed)
- `name`: String (descriptive name)
- `active`: Boolean (whether key is active)
- `created_at`: Date (creation timestamp)
- `last_used`: Date (last usage timestamp, nullable)

**Indexes:**
- `key_1`: Unique index on `key` field
- `active_1`: Index on `active` field
- `key_1_active_1`: Compound index for optimal query performance

## Scripts and Utilities

### 1. Database Initialization (`init.go`)

Initializes the database with required schema and sample data.

```bash
go run ./scripts/db/init.go
```

**Features:**
- Creates collections with proper indexes
- Generates sample API keys for testing
- Handles existing data gracefully
- Provides detailed logging

### 2. Migration Manager (`migrate.go`)

Handles database schema migrations and versioning.

```bash
go run ./scripts/db/migrate.go
```

**Features:**
- Version-controlled migrations
- Rollback capability
- Migration history tracking
- Safe migration execution

### 3. Database Setup Utility (`cmd/dbsetup`)

Comprehensive command-line utility for database management.

```bash
# Build the utility
go build -o bin/dbsetup ./cmd/dbsetup

# Run with options
./bin/dbsetup -help
./bin/dbsetup -all          # Full setup
./bin/dbsetup -init         # Initialize only
./bin/dbsetup -seed         # Seed data only
./bin/dbsetup -health       # Health check only
./bin/dbsetup -migrate      # Run migrations
./bin/dbsetup -rollback     # Rollback last migration
```

## Health Checks

The system includes comprehensive health monitoring:

### Health Check Endpoints

- `GET /health` - Overall system health
- `GET /health/live` - Liveness probe (for Kubernetes)
- `GET /health/ready` - Readiness probe (for Kubernetes)
- `GET /health/db` - Database-specific health

### Health Check Types

1. **Connectivity Check**: Tests basic MongoDB connection
2. **Operation Check**: Verifies database operations work
3. **Index Check**: Ensures required indexes exist
4. **Connection Pool Check**: Monitors connection pool health

### Example Health Response

```json
{
  "status": "healthy",
  "timestamp": "2024-01-15T10:30:00Z",
  "services": {
    "connectivity": {
      "service": "mongodb",
      "status": "healthy",
      "message": "all checks passed",
      "response_time": "5ms",
      "timestamp": "2024-01-15T10:30:00Z"
    },
    "connection_pool": {
      "service": "mongodb_pool",
      "status": "healthy",
      "message": "connection pool healthy: 2 current, 98 available",
      "response_time": "2ms",
      "timestamp": "2024-01-15T10:30:00Z"
    },
    "indexes": {
      "service": "mongodb_indexes",
      "status": "healthy",
      "message": "all required indexes present",
      "response_time": "3ms",
      "timestamp": "2024-01-15T10:30:00Z"
    }
  },
  "version": "1.0.0"
}
```

## Sample API Keys

The setup creates the following test API keys:

| Key | Name | Status | Purpose |
|-----|------|--------|---------|
| `test-api-key-1` | Test API Key 1 | Active | Basic testing |
| `test-api-key-2` | Test API Key 2 | Active | Multi-key testing |
| `inactive-test-key` | Inactive Test Key | Inactive | Testing inactive keys |
| `docker-dev-key` | Docker Development Key | Active | Docker development |
| Random keys | Generated Test Keys | Active | Load testing |

## Environment Configuration

Configure the database connection using environment variables:

```bash
# MongoDB connection
MONGODB_URI=mongodb://localhost:27017
MONGODB_DATABASE=solana_api
MONGODB_APIKEY_COLLECTION=api_keys
MONGODB_CONNECT_TIMEOUT=10s
MONGODB_MAX_POOL_SIZE=100
```

## Troubleshooting

### Common Issues

1. **Connection Failed**
   ```bash
   # Check if MongoDB is running
   make db-health
   
   # Verify connection string
   echo $MONGODB_URI
   ```

2. **Index Creation Failed**
   ```bash
   # Drop and recreate indexes
   make db-init
   ```

3. **Duplicate Key Errors**
   ```bash
   # Clear existing data and reseed
   # (Be careful in production!)
   ```

### Monitoring

Use the health check endpoints to monitor database status:

```bash
# Quick health check
curl http://localhost:8080/health

# Detailed database health
curl http://localhost:8080/health/db

# Check from command line
make db-health
```

## Production Considerations

### Security
- Use strong, unique API keys in production
- Enable MongoDB authentication
- Use TLS/SSL for database connections
- Regularly rotate API keys

### Performance
- Monitor connection pool usage
- Set appropriate timeouts
- Use read preferences for scaling
- Consider sharding for high load

### Backup
- Implement regular database backups
- Test backup restoration procedures
- Monitor backup success/failure

### Monitoring
- Set up alerts for health check failures
- Monitor database performance metrics
- Track API key usage patterns