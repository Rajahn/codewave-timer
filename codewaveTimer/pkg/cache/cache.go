package cache

import "time"

const defaultCap = 32

type Getter interface {
	Get(string) (interface{}, error)
}

type GetterFunc func(string) (interface{}, error)

func (g GetterFunc) Get(key string) (interface{}, error) {
	return g(key)
}

type CodewaveCache struct {
	shards    []*cacheShard
	hash      Hasher
	conf      Config
	shardMask uint64
	close     chan struct{}
}

// New initialize new instance e *CodewaveCache
func New(conf Config) (*CodewaveCache, error) {

	if conf.Cap <= 0 {
		conf.Cap = defaultCap
	}
	// init cache object
	cache := &CodewaveCache{
		shards:    make([]*cacheShard, conf.Shards),
		hash:      newDefaultHasher(),
		conf:      conf,
		shardMask: uint64(conf.Shards - 1),
		close:     make(chan struct{}),
	}
	var onRemove OnRemoveCallback
	if conf.OnRemoveWithReason != nil {
		onRemove = conf.OnRemoveWithReason
	} else {
		onRemove = cache.notProvidedOnRemove
	}

	// init shard
	for i := 0; i < conf.Shards; i++ {
		cache.shards[i] = newCacheShard(i, onRemove, cache.close)
	}
	return cache, nil
}

func NewCache(shards int) *CodewaveCache {
	// init cache object
	cache := &CodewaveCache{
		shards:    make([]*cacheShard, shards),
		hash:      newDefaultHasher(),
		shardMask: uint64(shards - 1),
		close:     make(chan struct{}),
	}
	var onRemove = cache.notProvidedOnRemove
	// init shard
	for i := 0; i < shards; i++ {
		cache.shards[i] = newCacheShard(i, onRemove, cache.close)
	}
	return cache
}

// Set add k/v or modify existing k/v
func (e *CodewaveCache) Set(key string, value interface{}, duration time.Duration) error {

	hashedKey := e.hash.Sum64(key)
	shard := e.getShard(hashedKey)
	return shard.set(key, value, duration)

}

// Get get k/v if exist,otherwise get an error
func (e *CodewaveCache) Get(key string) (interface{}, error) {
	hashedKey := e.hash.Sum64(key)
	shard := e.getShard(hashedKey)
	return shard.get(key)
}

// GetIfNotExist get an existing k/v  or  create a new k/v though by Getter
func (e *CodewaveCache) GetIfNotExist(key string, g Getter, duration time.Duration) (interface{}, error) {
	hashedKey := e.hash.Sum64(key)
	shard := e.getShard(hashedKey)
	return shard.getIfNotExist(key, g, duration)
}

func (e *CodewaveCache) GetOrSet(key string, value interface{}, duration time.Duration) (interface{}, error) {
	hashedKey := e.hash.Sum64(key)
	shard := e.getShard(hashedKey)
	return shard.getorset(key, value, duration)
}

func (e *CodewaveCache) Delete(key string) error {
	hashedKey := e.hash.Sum64(key)
	shard := e.getShard(hashedKey)
	return shard.del(key)
}

func (e *CodewaveCache) Count() int {
	count := 0
	for _, shard := range e.shards {
		count += shard.count()
	}
	return count
}

func (e *CodewaveCache) Foreach(f func(key string, value interface{})) {
	for _, shard := range e.shards {
		shard.foreach(f)
	}
}

func (e *CodewaveCache) Exists(key string) bool {
	hashedKey := e.hash.Sum64(key)
	shard := e.getShard(hashedKey)
	return shard.exists(key)
}

func (e *CodewaveCache) Close() error {
	close(e.close)
	return nil
}
func (e *CodewaveCache) getShard(hashedKey uint64) (shard *cacheShard) {
	return e.shards[hashedKey&e.shardMask]
}

func (e *CodewaveCache) notProvidedOnRemove(key string, value interface{}, reason RemoveReason) {
}
