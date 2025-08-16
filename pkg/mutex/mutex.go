package mutex

import (
	"sync"
	"time"
)

// RequestMutex provides per-address mutex locking to prevent duplicate concurrent requests
type RequestMutex struct {
	mutexes    map[string]*mutexEntry
	mapMutex   sync.RWMutex
	cleanupTTL time.Duration
	stopCh     chan struct{}
	stopped    bool
	stopMutex  sync.Mutex
}

// mutexEntry holds a mutex and its last access time for cleanup
type mutexEntry struct {
	mutex      *sync.Mutex
	lastAccess time.Time
}

// New creates a new RequestMutex instance with automatic cleanup
func New(cleanupTTL time.Duration) *RequestMutex {
	rm := &RequestMutex{
		mutexes:    make(map[string]*mutexEntry),
		cleanupTTL: cleanupTTL,
		stopCh:     make(chan struct{}),
	}

	// Start cleanup goroutine
	go rm.cleanup()

	return rm
}

// GetMutex returns a mutex for the given address, creating one if it doesn't exist
func (rm *RequestMutex) GetMutex(address string) *sync.Mutex {
	rm.mapMutex.RLock()
	entry, exists := rm.mutexes[address]
	if exists {
		// Update last access time
		entry.lastAccess = time.Now()
		rm.mapMutex.RUnlock()
		return entry.mutex
	}
	rm.mapMutex.RUnlock()

	// Need to create a new mutex
	rm.mapMutex.Lock()
	defer rm.mapMutex.Unlock()

	// Double-check in case another goroutine created it
	if entry, exists := rm.mutexes[address]; exists {
		entry.lastAccess = time.Now()
		return entry.mutex
	}

	// Create new mutex entry
	newEntry := &mutexEntry{
		mutex:      &sync.Mutex{},
		lastAccess: time.Now(),
	}
	rm.mutexes[address] = newEntry

	return newEntry.mutex
}

// Lock locks the mutex for the given address
func (rm *RequestMutex) Lock(address string) {
	mutex := rm.GetMutex(address)
	mutex.Lock()
}

// Unlock unlocks the mutex for the given address
func (rm *RequestMutex) Unlock(address string) {
	rm.mapMutex.RLock()
	entry, exists := rm.mutexes[address]
	rm.mapMutex.RUnlock()

	if exists {
		entry.mutex.Unlock()
	}
}

// Size returns the number of mutexes currently stored
func (rm *RequestMutex) Size() int {
	rm.mapMutex.RLock()
	defer rm.mapMutex.RUnlock()
	return len(rm.mutexes)
}

// cleanup runs periodically to remove unused mutexes to prevent memory leaks
func (rm *RequestMutex) cleanup() {
	ticker := time.NewTicker(rm.cleanupTTL)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rm.removeUnused()
		case <-rm.stopCh:
			return
		}
	}
}

// removeUnused removes mutexes that haven't been accessed recently
func (rm *RequestMutex) removeUnused() {
	rm.mapMutex.Lock()
	defer rm.mapMutex.Unlock()

	now := time.Now()
	for address, entry := range rm.mutexes {
		// Only remove if mutex is not locked and hasn't been accessed recently
		if now.Sub(entry.lastAccess) > rm.cleanupTTL {
			// Try to lock the mutex to ensure it's not in use
			if entry.mutex.TryLock() {
				entry.mutex.Unlock()
				delete(rm.mutexes, address)
			}
		}
	}
}

// Stop stops the cleanup goroutine
func (rm *RequestMutex) Stop() {
	rm.stopMutex.Lock()
	defer rm.stopMutex.Unlock()

	if !rm.stopped {
		rm.stopped = true
		close(rm.stopCh)
	}
}
