# Request Mutex Package

The `mutex` package provides per-address mutex locking to prevent duplicate concurrent requests for the same resource (e.g., wallet address). This is particularly useful in scenarios where multiple concurrent requests for the same resource should be deduplicated to avoid unnecessary work.

## Features

- **Per-address locking**: Each unique address gets its own mutex
- **Automatic cleanup**: Unused mutexes are automatically cleaned up to prevent memory leaks
- **Thread-safe**: All operations are safe for concurrent use
- **Memory efficient**: Mutexes are only created when needed and cleaned up when unused

## Usage

### Basic Usage

```go
package main

import (
    "time"
    "solana-balance-api/pkg/mutex"
)

func main() {
    // Create a new RequestMutex with 5-minute cleanup TTL
    rm := mutex.New(5 * time.Minute)
    defer rm.Stop()

    walletAddress := "9WzDXwBbmkg8ZTbNMqUxvQRAyrZzDsGYdLVL9zYtAWWM"

    // Method 1: Get mutex directly
    mutex := rm.GetMutex(walletAddress)
    mutex.Lock()
    // ... do work ...
    mutex.Unlock()

    // Method 2: Use convenience methods
    rm.Lock(walletAddress)
    // ... do work ...
    rm.Unlock(walletAddress)
}
```

### Concurrent Request Deduplication

```go
func handleBalanceRequest(rm *mutex.RequestMutex, address string) {
    // Lock for this specific address
    rm.Lock(address)
    defer rm.Unlock(address)

    // Check cache first
    if balance, found := cache.Get(address); found {
        return balance
    }

    // Only one goroutine per address will reach here
    balance := fetchFromRPC(address)
    cache.Set(address, balance)
    return balance
}
```

## API Reference

### Types

#### RequestMutex

```go
type RequestMutex struct {
    // private fields
}
```

The main struct that manages per-address mutexes.

### Functions

#### New

```go
func New(cleanupTTL time.Duration) *RequestMutex
```

Creates a new RequestMutex instance with automatic cleanup. The `cleanupTTL` parameter determines how long unused mutexes are kept before being cleaned up.

### Methods

#### GetMutex

```go
func (rm *RequestMutex) GetMutex(address string) *sync.Mutex
```

Returns a mutex for the given address. Creates a new mutex if one doesn't exist for the address.

#### Lock

```go
func (rm *RequestMutex) Lock(address string)
```

Convenience method to lock the mutex for the given address.

#### Unlock

```go
func (rm *RequestMutex) Unlock(address string)
```

Convenience method to unlock the mutex for the given address.

#### Size

```go
func (rm *RequestMutex) Size() int
```

Returns the number of mutexes currently stored. Useful for monitoring and testing.

#### Stop

```go
func (rm *RequestMutex) Stop()
```

Stops the cleanup goroutine. Should be called when the RequestMutex is no longer needed.

## Memory Management

The RequestMutex automatically cleans up unused mutexes to prevent memory leaks:

- Mutexes that haven't been accessed for longer than the `cleanupTTL` are eligible for removal
- Only unlocked mutexes are removed during cleanup
- The cleanup process runs periodically in a background goroutine

## Thread Safety

All operations are thread-safe and can be called concurrently from multiple goroutines:

- Multiple goroutines can safely call `GetMutex()` for the same or different addresses
- The internal mutex map is protected by a read-write mutex
- Cleanup operations are synchronized with regular operations

## Performance Considerations

- **Read-heavy workload**: Uses `sync.RWMutex` for the internal map to allow concurrent reads
- **Memory efficient**: Mutexes are created on-demand and cleaned up automatically
- **Lock contention**: Only requests for the same address will contend for the same mutex

## Testing

Run the tests with:

```bash
go test ./pkg/mutex/...
```

Run benchmarks with:

```bash
go test -bench=. ./pkg/mutex/
```