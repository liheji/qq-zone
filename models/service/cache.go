package service

import (
	"github.com/patrickmn/go-cache"
	"sync"
	"time"
)

var (
	mutexCache   sync.Mutex
	cacheService *cache.Cache
)

const (
	RETRY_TIMES    = 30
	DEFAULT_EXPIRE = 24 * time.Hour
)

// SingleCache 获取 cache.Cache 实例
func SingleCache() *cache.Cache {
	if cacheService == nil {
		mutexCache.Lock()
		if cacheService == nil {
			cacheService = cache.New(DEFAULT_EXPIRE, 1*time.Minute)
		}
		mutexCache.Unlock()
	}
	return cacheService
}

func CheckCache(key string) bool {
	err := SingleCache().Increment(key, 1)
	if err != nil {
		return false
	}
	if val, exit := SingleCache().Get(key); exit {
		if val.(int) <= RETRY_TIMES {
			return true
		}
	}
	return false
}
