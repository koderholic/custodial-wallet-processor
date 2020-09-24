package cache

import (
	"time"

	"github.com/patrickmn/go-cache"
)

// MemoryCache ...
type Memory struct {
	Cache *cache.Cache
}

// InitializeCache ...
func Initialize(expiry time.Duration, purgeInterval time.Duration) *Memory {
	newCache := cache.New(expiry, purgeInterval)
	memoryCache := Memory{
		Cache: newCache,
	}
	return &memoryCache
}

// Set ...
func (memory *Memory) Set(key string, value interface{}, expiry bool) {
	if expiry {
		memory.Cache.Set(key, value, cache.DefaultExpiration)
	} else {
		memory.Cache.Set(key, value, cache.NoExpiration)
	}
}

// Get ...
func (memory *Memory) Get(key string) interface{} {
	cacheValue, _ := memory.Cache.Get(key)
	return cacheValue
}
