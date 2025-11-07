package cache

import (
	"sync"
	"time"
)

// Cache represents a simple in-memory cache with TTL
type Cache struct {
	items    map[string]*cacheItem
	mu       sync.RWMutex
	ttl      time.Duration
	maxSize  int
}

type cacheItem struct {
	value      interface{}
	expiresAt  time.Time
	accessTime time.Time
}

// NewCache creates a new cache instance
func NewCache(ttl time.Duration, maxSize int) *Cache {
	cache := &Cache{
		items:   make(map[string]*cacheItem),
		ttl:     ttl,
		maxSize: maxSize,
	}

	// Start cleanup goroutine
	go cache.cleanup()

	return cache
}

// Get retrieves a value from the cache
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, exists := c.items[key]
	if !exists {
		return nil, false
	}

	// Check if expired
	if time.Now().After(item.expiresAt) {
		return nil, false
	}

	// Update access time
	item.accessTime = time.Now()

	return item.value, true
}

// Set stores a value in the cache
func (c *Cache) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check max size and evict LRU if needed
	if len(c.items) >= c.maxSize {
		c.evictLRU()
	}

	c.items[key] = &cacheItem{
		value:      value,
		expiresAt:  time.Now().Add(c.ttl),
		accessTime: time.Now(),
	}
}

// Delete removes a key from the cache
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
}

// Clear removes all items from the cache
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[string]*cacheItem)
}

// evictLRU evicts the least recently used item
func (c *Cache) evictLRU() {
	var oldestKey string
	var oldestTime time.Time
	first := true

	for key, item := range c.items {
		if first || item.accessTime.Before(oldestTime) {
			oldestKey = key
			oldestTime = item.accessTime
			first = false
		}
	}

	if oldestKey != "" {
		delete(c.items, oldestKey)
	}
}

// cleanup periodically removes expired items
func (c *Cache) cleanup() {
	ticker := time.NewTicker(c.ttl / 2)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, item := range c.items {
			if now.After(item.expiresAt) {
				delete(c.items, key)
			}
		}
		c.mu.Unlock()
	}
}

// Size returns the number of items in the cache
func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}





