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

var KeyNotFoundError = errors.New("key not found")

type Cache interface {
	Set(key, value interface{}) error
	SetWithExpire(key, value interface{}, expiration time.Duration) error
	GetIFPresent(key interface{}) (interface{}, error)
	GetALL(checkExpired bool) map[interface{}]interface{}
	get(key interface{}, onLoad bool) (interface{}, error)
	Remove(key interface{}) bool
	Purge()
	Keys(checkExpired bool) []interface{}
	Len(checkExpired bool) int
	Existed(key interface{}) bool

	statsAccessor
}

type LoadingCache interface {
	Cache
	Get(ctx context.Context, key interface{}) (interface{}, error)
	Refresh(ctx context.Context, key interface{}) (interface{}, error)
}

type baseCache struct {
	clock            Clock
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
	clock            Clock
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
		clock: NewRealClock(),
		tp:    TypeSimple,
		size:  size,
	}
}

func (cb *CacheBuilder) Clock(clock Clock) *CacheBuilder {
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

func buildCache(c *baseCache, cb *CacheBuilder) {
	c.clock = cb.clock
	c.size = cb.size
	c.loaderExpireFunc = cb.loaderExpireFunc
	c.expiration = cb.expiration
	c.addedFunc = cb.addedFunc
	c.deserializeFunc = cb.deserializeFunc
	c.serializeFunc = cb.serializeFunc
	c.evictedFunc = cb.evictedFunc
	c.purgeVisitorFunc = cb.purgeVisitorFunc
	c.stats = &stats{}
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

// load a new value using by specified key.
func (c *baseCache) Refresh(ctx context.Context, key interface{}) (interface{}, error) {
	panic("to to")
}
