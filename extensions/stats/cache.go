package stats

import (
	"sync"
	"time"
)

// CacheService provides caching for frequently accessed stats
type CacheService struct {
	cache map[string]*cacheEntry
	mu    sync.RWMutex
	ttl   time.Duration
}

type cacheEntry struct {
	data      interface{}
	expiresAt time.Time
}

// NewCacheService creates a new cache service
func NewCacheService(ttl time.Duration) *CacheService {
	service := &CacheService{
		cache: make(map[string]*cacheEntry),
		ttl:   ttl,
	}
	
	// Start cleanup goroutine
	go service.cleanup()
	
	return service
}

// Get retrieves a value from cache
func (c *CacheService) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	entry, exists := c.cache[key]
	if !exists || time.Now().After(entry.expiresAt) {
		return nil, false
	}
	
	return entry.data, true
}

// Set stores a value in cache
func (c *CacheService) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.cache[key] = &cacheEntry{
		data:      value,
		expiresAt: time.Now().Add(c.ttl),
	}
}

// Delete removes a value from cache
func (c *CacheService) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	delete(c.cache, key)
}

// cleanup periodically removes expired entries
func (c *CacheService) cleanup() {
	ticker := time.NewTicker(c.ttl)
	defer ticker.Stop()
	
	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, entry := range c.cache {
			if now.After(entry.expiresAt) {
				delete(c.cache, key)
			}
		}
		c.mu.Unlock()
	}
}