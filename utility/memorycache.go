package utility

import (
	"time"

	"github.com/patrickmn/go-cache"
)

// MemoryCache ...
type MemoryCache struct {
	Cache *cache.Cache
}

// InitializeCache ...
func InitializeCache(expiry time.Duration, purgeInterval time.Duration) *MemoryCache {
	newCache := cache.New(expiry, purgeInterval)
	memoryCache := MemoryCache{
		Cache: newCache,
	}
	return &memoryCache
}

// Set ...
func (memory *MemoryCache) Set(key string, value interface{}, expiry bool) {
	if expiry {
		memory.Cache.Set(key, value, cache.DefaultExpiration)
	} else {
		memory.Cache.Set(key, value, cache.NoExpiration)
	}
}

// Get ...
func (memory *MemoryCache) Get(key string) interface{} {
	cacheValue, _ := memory.Cache.Get(key)
	return cacheValue
}
