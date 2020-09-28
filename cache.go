package gcache

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

const (
	TypeSimple = "simple"
	TypeLru    = "lru"
	TypeLfu    = "lfu"
	TypeArc    = "arc"
)

// ErrKeyNotFound return error if key not found or expired
var ErrKeyNotFound = errors.New("key not found")

type Cache interface {
	// Set a new key-value pair
	Set(key, value interface{}) error

	// SetWithExpire Set a new key-value pair with an expiration time
	SetWithExpire(key, value interface{}, expiration time.Duration) error

	// GetIFPresent gets a value from cache pool using key if it exists.
	// If it dose not exists key, returns ErrKeyNotFound.
	// And send a request which refresh value for specified key if cache object has LoaderFunc.
	GetIFPresent(key interface{}) (interface{}, error)

	// GetALL returns all key-value pairs in the cache.
	GetALL(checkExpired bool) map[interface{}]interface{}

	// Remove removes the provided key from the cache.
	Remove(key interface{}) bool

	// Completely clear the cache
	Purge()

	// Keys returns a slice of the keys in the cache.
	Keys(checkExpired bool) []interface{}

	// Len returns the number of items in the cache.
	Len(checkExpired bool) int

	//Existed checks if key exists in cache
	Existed(key interface{}) bool

	set(key, value interface{}) (interface{}, error)
	get(key interface{}, onLoad bool) (interface{}, error)

	statsAccessor
}

type LoadingCache interface {
	Cache
	// Get a value from cache pool using key if it exists. If not exists and it has LoaderFunc,
	// it will generate the value using you have specified LoaderFunc method returns value.
	Get(ctx context.Context, key interface{}) (interface{}, error)

	//Refresh refresh a new value using by specified key.
	Refresh(ctx context.Context, key interface{}) (interface{}, error)
}

type (
	LoaderFunc       func(context.Context, interface{}) (interface{}, error)
	LoaderExpireFunc func(context.Context, interface{}) (interface{}, *time.Duration, error)
	EvictedFunc      func(interface{}, interface{})
	PurgeVisitorFunc func(interface{}, interface{})
	AddedFunc        func(interface{}, interface{})
	DeserializeFunc  func(interface{}, interface{}) (interface{}, error)
	SerializeFunc    func(interface{}, interface{}) (interface{}, error)
)

type CacheBuilder struct {
	clock            clock
	tp               string
	size             int
	loaderExpireFunc LoaderExpireFunc
	evictedFunc      EvictedFunc
	purgeVisitorFunc PurgeVisitorFunc
	addedFunc        AddedFunc
	expiration       *time.Duration
	deserializeFunc  DeserializeFunc
	serializeFunc    SerializeFunc
}

func New(size int) *CacheBuilder {
	return &CacheBuilder{
		clock: newRealClock(),
		tp:    TypeSimple,
		size:  size,
	}
}

func (cb *CacheBuilder) Clock(clock clock) *CacheBuilder {
	cb.clock = clock
	return cb
}

// Set a loader function.
// loaderFunc: create a new value with this function if cached value is expired.
func (cb *CacheBuilder) LoaderFunc(loaderFunc LoaderFunc) *loadingCacheBuilder {
	cb.loaderExpireFunc = func(ctx context.Context, k interface{}) (interface{}, *time.Duration, error) {
		v, err := loaderFunc(ctx, k)
		return v, nil, err
	}
	return &loadingCacheBuilder{CacheBuilder: cb}
}

// Set a loader function with expiration.
// loaderExpireFunc: create a new value with this function if cached value is expired.
// If nil returned instead of time.Duration from loaderExpireFunc than value will never expire.
func (cb *CacheBuilder) LoaderExpireFunc(loaderExpireFunc LoaderExpireFunc) *loadingCacheBuilder {
	cb.loaderExpireFunc = loaderExpireFunc
	return &loadingCacheBuilder{CacheBuilder: cb}
}

func (cb *CacheBuilder) EvictType(tp string) *CacheBuilder {
	cb.tp = tp
	return cb
}

func (cb *CacheBuilder) Simple() *CacheBuilder {
	return cb.EvictType(TypeSimple)
}

func (cb *CacheBuilder) LRU() *CacheBuilder {
	return cb.EvictType(TypeLru)
}

func (cb *CacheBuilder) LFU() *CacheBuilder {
	return cb.EvictType(TypeLfu)
}

func (cb *CacheBuilder) ARC() *CacheBuilder {
	return cb.EvictType(TypeArc)
}

func (cb *CacheBuilder) EvictedFunc(evictedFunc EvictedFunc) *CacheBuilder {
	cb.evictedFunc = evictedFunc
	return cb
}

func (cb *CacheBuilder) PurgeVisitorFunc(purgeVisitorFunc PurgeVisitorFunc) *CacheBuilder {
	cb.purgeVisitorFunc = purgeVisitorFunc
	return cb
}

func (cb *CacheBuilder) AddedFunc(addedFunc AddedFunc) *CacheBuilder {
	cb.addedFunc = addedFunc
	return cb
}

func (cb *CacheBuilder) DeserializeFunc(deserializeFunc DeserializeFunc) *CacheBuilder {
	cb.deserializeFunc = deserializeFunc
	return cb
}

func (cb *CacheBuilder) SerializeFunc(serializeFunc SerializeFunc) *CacheBuilder {
	cb.serializeFunc = serializeFunc
	return cb
}

func (cb *CacheBuilder) Expiration(expiration time.Duration) *CacheBuilder {
	cb.expiration = &expiration
	return cb
}

func (cb *CacheBuilder) Build() Cache {
	if cb.size <= 0 && cb.tp != TypeSimple {
		panic("gcache: Cache size <= 0")
	}

	return cb.build()
}

func (cb *CacheBuilder) build() LoadingCache {
	switch cb.tp {
	case TypeSimple:
		return newSimpleCache(cb)
	case TypeLru:
		return newLRUCache(cb)
	case TypeLfu:
		return newLFUCache(cb)
	case TypeArc:
		return newARC(cb)
	default:
		panic("gcache: Unknown type " + cb.tp)
	}
}

type loadingCacheBuilder struct {
	*CacheBuilder
}

func (cb *loadingCacheBuilder) EvictType(tp string) *loadingCacheBuilder {
	cb.tp = tp
	return cb
}

func (cb *loadingCacheBuilder) Simple() *loadingCacheBuilder {
	return cb.EvictType(TypeSimple)
}

func (cb *loadingCacheBuilder) LRU() *loadingCacheBuilder {
	return cb.EvictType(TypeLru)
}

func (cb *loadingCacheBuilder) LFU() *loadingCacheBuilder {
	return cb.EvictType(TypeLfu)
}

func (cb *loadingCacheBuilder) ARC() *loadingCacheBuilder {
	return cb.EvictType(TypeArc)
}

func (cb *loadingCacheBuilder) EvictedFunc(evictedFunc EvictedFunc) *loadingCacheBuilder {
	cb.evictedFunc = evictedFunc
	return cb
}

func (cb *loadingCacheBuilder) PurgeVisitorFunc(purgeVisitorFunc PurgeVisitorFunc) *loadingCacheBuilder {
	cb.purgeVisitorFunc = purgeVisitorFunc
	return cb
}

func (cb *loadingCacheBuilder) AddedFunc(addedFunc AddedFunc) *loadingCacheBuilder {
	cb.addedFunc = addedFunc
	return cb
}

func (cb *loadingCacheBuilder) DeserializeFunc(deserializeFunc DeserializeFunc) *loadingCacheBuilder {
	cb.deserializeFunc = deserializeFunc
	return cb
}

func (cb *loadingCacheBuilder) SerializeFunc(serializeFunc SerializeFunc) *loadingCacheBuilder {
	cb.serializeFunc = serializeFunc
	return cb
}

func (cb *loadingCacheBuilder) Expiration(expiration time.Duration) *loadingCacheBuilder {
	cb.expiration = &expiration
	return cb
}

func (cb *loadingCacheBuilder) Build() LoadingCache {
	if cb.loaderExpireFunc == nil {
		panic("loader func required")
	}
	return cb.CacheBuilder.Build().(LoadingCache)
}

func buildCache(b *baseCache, c Cache, cb *CacheBuilder) {
	b.cache = c

	b.clock = cb.clock
	b.size = cb.size
	b.loaderExpireFunc = cb.loaderExpireFunc
	b.expiration = cb.expiration
	b.addedFunc = cb.addedFunc
	b.deserializeFunc = cb.deserializeFunc
	b.serializeFunc = cb.serializeFunc
	b.evictedFunc = cb.evictedFunc
	b.purgeVisitorFunc = cb.purgeVisitorFunc
	b.stats = &stats{}
}

type cacheItem struct {
	clock      clock
	key        interface{}
	value      interface{}
	expiration *time.Time
}

// IsExpired returns boolean value whether this item is expired or not.
func (item *cacheItem) IsExpired(now *time.Time) bool {
	if item.expiration == nil {
		return false
	}
	if now == nil {
		t := item.clock.Now()
		now = &t
	}
	return item.expiration.Before(*now)
}

type baseCache struct {
	cache Cache

	clock            clock
	size             int
	loaderExpireFunc LoaderExpireFunc
	evictedFunc      EvictedFunc
	purgeVisitorFunc PurgeVisitorFunc
	addedFunc        AddedFunc
	deserializeFunc  DeserializeFunc
	serializeFunc    SerializeFunc
	expiration       *time.Duration
	mu               sync.RWMutex
	loadGroup        Group
	*stats
}

func (c *baseCache) Set(key, value interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, err := c.cache.set(key, value)
	return err
}

func (c *baseCache) SetWithExpire(key, value interface{}, expiration time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	item, err := c.cache.set(key, value)
	if err != nil {
		return err
	}

	t := c.clock.Now().Add(expiration)
	item.(*cacheItem).expiration = &t
	return nil
}

// Get a value from cache pool using key if it exists. If not exists and it has LoaderFunc, it will generate the value using you have specified LoaderFunc method returns value.
func (c *baseCache) Get(ctx context.Context, key interface{}) (interface{}, error) {
	v, err := c.cache.get(key, false)
	if err == ErrKeyNotFound {
		return c.getWithLoader(ctx, key, true)
	}
	return v, err
}

// GetIFPresent gets a value from cache pool using key if it exists.
// If it dose not exists key, returns ErrKeyNotFound.
// And send a request which refresh value for specified key if cache object has LoaderFunc.
func (c *baseCache) GetIFPresent(key interface{}) (interface{}, error) {
	v, err := c.cache.get(key, false)
	if err == ErrKeyNotFound {
		return c.getWithLoader(context.Background(), key, false)
	}
	return v, nil
}

// load a new value using by specified key.
func (c *baseCache) load(ctx context.Context, key interface{}, cb func(interface{}, *time.Duration, error) (interface{}, error), isWait bool) (interface{}, bool, error) {
	v, called, err := c.loadGroup.Do(key, func() (v interface{}, e error) {
		defer func() {
			if r := recover(); r != nil {
				e = fmt.Errorf("Loader panics: %v", r)
			}
		}()
		return cb(c.loaderExpireFunc(ctx, key))
	}, isWait)
	if err != nil {
		return nil, called, err
	}
	return v, called, nil
}

func (c *baseCache) getWithLoader(ctx context.Context, key interface{}, isWait bool) (interface{}, error) {
	if c.loaderExpireFunc == nil {
		return nil, ErrKeyNotFound
	}
	value, _, err := c.load(ctx, key, func(v interface{}, expiration *time.Duration, e error) (interface{}, error) {
		if e != nil {
			return nil, e
		}
		c.mu.Lock()
		defer c.mu.Unlock()
		item, err := c.cache.set(key, v)
		if err != nil {
			return nil, err
		}
		if expiration != nil {
			t := c.clock.Now().Add(*expiration)
			item.(*cacheItem).expiration = &t
		}
		return v, nil
	}, isWait)
	if err != nil {
		return nil, err
	}
	return value, nil
}

// load a new value using by specified key.
func (c *baseCache) Refresh(ctx context.Context, key interface{}) (interface{}, error) {
	return c.getWithLoader(ctx, key, true)
}
