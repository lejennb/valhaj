package memory

import (
	"crypto/sha1"
	"sync"

	"lj.com/valhaj/internal/config"
)

var (
	Cache     ShardedCache
	Container CacheContainer
)

type shard struct {
	sync.RWMutex
	m map[string]string
}

type ShardedCache []*shard

func NewShardedCache(shardCount int) ShardedCache {
	shards := make([]*shard, shardCount)

	for i := 0; i < shardCount; i++ {
		shards[i] = &shard{
			m: make(map[string]string),
		}
	}

	return shards
}

type CacheContainer []*ShardedCache

func NewCacheContainer(cacheCount, shardCount int) CacheContainer {
	caches := make([]*ShardedCache, cacheCount)

	for i := 0; i < cacheCount; i++ {
		cache := NewShardedCache(shardCount)
		caches[i] = &cache
	}

	return caches
}

/* shard ops */

func (sc ShardedCache) getShardIndex(key string) int {
	checksum := sha1.Sum([]byte(key))
	hash := int(checksum[17])
	return hash % len(sc)
}

func (sc ShardedCache) getShard(key string) *shard {
	index := sc.getShardIndex(key)
	return sc[index]
}

/* map ops */

func (sc ShardedCache) Load(key string) (string, bool) {
	shard := sc.getShard(key)
	shard.RLock()
	defer shard.RUnlock()

	value, ok := shard.m[key]
	return value, ok
}

func (sc ShardedCache) LoadAndDelete(key string) (string, bool) {
	shard := sc.getShard(key)
	shard.Lock()
	defer shard.Unlock()

	value, ok := shard.m[key]
	if ok {
		delete(shard.m, key)
	}
	return value, ok
}

func (sc ShardedCache) LoadExistStore(key, value string, exists, overwrite bool) (string, bool) {
	shard := sc.getShard(key)
	shard.Lock()
	defer shard.Unlock()

	oldValue, ok := shard.m[key]
	if ok == exists || overwrite {
		shard.m[key] = value
	}
	return oldValue, ok
}

func (sc ShardedCache) LoadModifyStore(key string, modifier func(string) (string, bool), initial string) (string, bool) {
	shard := sc.getShard(key)
	shard.Lock()
	defer shard.Unlock()

	value, ok := shard.m[key]
	if !ok {
		value = initial
	}
	value, ok = modifier(value)
	shard.m[key] = value
	return value, ok
}

func (sc ShardedCache) Store(key, value string) {
	shard := sc.getShard(key)
	shard.Lock()
	defer shard.Unlock()

	shard.m[key] = value
}

func (sc ShardedCache) Delete(key string) {
	shard := sc.getShard(key)
	shard.Lock()
	defer shard.Unlock()

	delete(shard.m, key)
}

func (sc ShardedCache) Range() ([]string, int) {
	var items []string
	var total int
	for _, shard := range sc {
		shard.Lock()
		for key, value := range shard.m {
			items = append(items, key, value)
		}
		shard.Unlock()
	}
	total = len(items) / 2

	return items, total
}

func (sc ShardedCache) Count() (int, []int) {
	var total int
	var subtotal = make([]int, 0, config.MemoryCacheShardCount)
	var shardMapSize int
	for _, shard := range sc {
		shard.RLock()
		shardMapSize = len(shard.m)
		shard.RUnlock()

		total += shardMapSize
		subtotal = append(subtotal, shardMapSize)
	}

	return total, subtotal
}

func (sc ShardedCache) Clear() {
	for _, shard := range sc {
		shard.Lock()
		shard.m = nil
		shard.m = make(map[string]string)
		shard.Unlock()
	}
}
