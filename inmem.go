package httpcache

import "sync"

// InMemoryCache is an implementation of Cache that stores responses in an in-memory
// map. This cache if volatile and will be cleared when the program exits, but is often
// a good choice for testing or short-lived applications.
type InMemoryCache struct {
	sync.RWMutex
	store map[string][]byte
}

// Get the []byte representation of the response and true if present.
func (c *InMemoryCache) Get(key string) (val []byte, ok bool) {
	c.RLock()
	val, ok = c.store[key]
	c.RUnlock()
	return
}

// Put stores the []byte representation of the response with the specified key.
func (c *InMemoryCache) Put(key string, val []byte) {
	c.Lock()
	if c.store == nil {
		c.store = make(map[string][]byte)
	}
	c.store[key] = val
	c.Unlock()
}

// Rm removes the cached response associated with the key.
func (c *InMemoryCache) Rm(key string) {
	c.Lock()
	delete(c.store, key)
	c.Unlock()
}
