/*
Package ristretto provides a fast, concurrent implementation of httpcache.Cache built
with a focus on performance and correctness using the github.com/dgraph-io/ristretto
library as the underlying storage.

This backend is suitable for applications that need to cache millions of entries in high
throughput environments with hundreds of threads accessing the cache concurrently.

Example Usage:

	cache := ristretto.New(&ristretto.Config{
		NumCounters: 1e7,     // number of keys to track frequency of (10M).
		MaxCost:     1 << 30, // maximum cost of cache (1GB).
		BufferItems: 64,      // number of keys per Get buffer.
	})

	transport := httpcache.NewTransport(cache)
	client := transport.Client()

	// Later ...
	cache.Close()
*/
package ristretto

import (
	"io"

	"github.com/dgraph-io/ristretto/v2"
	"go.rtnl.ai/httpcache"
)

type Cache struct {
	cache *ristretto.Cache[string, []byte]
}

var _ httpcache.Cache = (*Cache)(nil)
var _ io.Closer = (*Cache)(nil)

// Create a new Ristretto-backed httpcache.Cache with the specified configuration.
func New(config *Config) (_ *Cache, err error) {
	cache := &Cache{}
	if cache.cache, err = ristretto.NewCache(config.convert()); err != nil {
		return nil, err
	}

	return cache, nil
}

// Get returns the value (if any) and a boolean representing whether the value was found
// or not. The value can be nil and the boolean can be true at the same time. Get will
// not return expired items.
func (c *Cache) Get(key string) ([]byte, bool) {
	return c.cache.Get(key)
}

// Put attempts to add the key-value item to the cache. If the cache has reached the
// maximum size, it may evict other items to make room for the new item. Put does not
// set an explicitly cost for the item; instead, it relies on the Cost function defined
// in the Config to determine the cost of the item. If using a dynamic Cost function,
// it is possible that the item may be dropped and not cached rather than evicting other
// higher value items.
//
// Be careful when modifying the value byte slice after calling Put, calling `append`
// may update the underlying array pointer which will not be reflected in the cache.
func (c *Cache) Put(key string, value []byte) {
	c.cache.Set(key, value, 0)
}

// Del deletes the key-value item from the cache if it exists.
func (c *Cache) Del(key string) {
	c.cache.Del(key)
}

// Close stops all goroutines and closes all channels.
// Implements io.Closer.
func (c *Cache) Close() error {
	c.cache.Close()
	return nil
}

// Wait blocks until all buffered writes have been applied.
// This ensures a call to Put() will be visible to future calls to Get().
func (c *Cache) Wait() {
	c.cache.Wait()
}
