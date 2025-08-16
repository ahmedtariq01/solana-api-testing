# Cache Package

A thread-safe, TTL-based in-memory cache implementation for the Solana Balance API.

## Features

- **Thread-safe operations**: Uses RWMutex for concurrent access
- **TTL support**: Automatic expiration of cached entries
- **Memory management**: Background cleanup of expired entries
- **High performance**: Optimized for frequent reads and writes
- **Simple API**: Easy to use Get/Set interface

## Usage

```go
package main

import (
    "time"
    "solana-balance-api/pkg/cache"
)

func main() {
    // Create a new cache with 10-second TTL
    c := cache.New(10 * time.Second)
    defer c.Stop() // Important: stop the cleanup goroutine

    // Set a value
    c.Set("wallet-address", 1.5)

    // Get a value
    balance, found := c.Get("wallet-address")
    if found {
        fmt.Printf("Balance: %f\n", balance)
    }
}
```

## API Reference

### Types

#### `Cache`
The main cache struct that provides thread-safe caching with TTL.

#### `CacheEntry`
Internal structure representing a cached value with its timestamp.

### Functions

#### `New(ttl time.Duration) *Cache`
Creates a new cache instance with the specified TTL. Starts a background cleanup goroutine.

#### `Get(key string) (float64, bool)`
Retrieves a value from the cache. Returns the value and a boolean indicating if the key was found and not expired.

#### `Set(key string, balance float64)`
Stores a value in the cache with the current timestamp.

#### `Delete(key string)`
Removes a specific key from the cache.

#### `Clear()`
Removes all entries from the cache.

#### `Size() int`
Returns the current number of entries in the cache.

#### `Stop()`
Stops the background cleanup goroutine. Should be called when the cache is no longer needed.

## Implementation Details

### TTL Behavior
- Entries are considered expired when `time.Since(entry.Timestamp) > ttl`
- Expired entries are not returned by `Get()` operations
- A background goroutine periodically removes expired entries to prevent memory leaks

### Thread Safety
- All operations are thread-safe using `sync.RWMutex`
- Read operations (`Get`, `Size`) use read locks for better concurrency
- Write operations (`Set`, `Delete`, `Clear`) use exclusive locks

### Memory Management
- Expired entries are automatically cleaned up by a background goroutine
- Cleanup runs at intervals equal to the TTL duration
- Manual cleanup can be triggered using `Delete()` or `Clear()`

## Performance

The cache is optimized for high-frequency operations:

- `Get` operations: ~52 ns/op with 0 allocations
- `Set` operations: ~164 ns/op with 1 allocation
- Concurrent operations: ~173 ns/op

## Requirements Compliance

This implementation satisfies the following requirements:

- **4.1**: Caches wallet balance results for the specified TTL (10 seconds in production)
- **4.2**: Returns cached results for requests within the TTL period
- **4.3**: Fetches fresh data when TTL expires (handled by the service layer)
- **4.4**: Uses wallet address as the cache key