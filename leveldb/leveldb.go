package leveldb

import (
	"errors"
	"log/slog"

	"github.com/syndtr/goleveldb/leveldb"
	"go.rtnl.ai/httpcache"
)

// Cache is an implementation of httpcache.Cache with leveldb storage
type Cache struct {
	db *leveldb.DB
}

// New returns a cache that will store cached data in a leveldb database at the path.
func New(path string) (_ *Cache, err error) {
	cache := &Cache{}
	if cache.db, err = leveldb.OpenFile(path, nil); err != nil {
		return nil, err
	}
	return cache, nil
}

// Make returns a cache using the specified db instance as the underlying storage.
func Make(db *leveldb.DB) *Cache {
	return &Cache{db: db}
}

// Get a value from the cache for the specified key. If any error other than
// ErrNotFound occurs it is logged and false is returned.
func (c *Cache) Get(key string) ([]byte, bool) {
	data, err := c.db.Get([]byte(key), nil)
	if err != nil {
		if !errors.Is(err, leveldb.ErrNotFound) {
			httpcache.GetLogger().Warn("failed to read from leveldb cache", slog.Any("error", err))
		}
		return nil, false
	}
	return data, true
}

// Put a value into the cache with the specified key. If an error occurs it is logged.
func (c *Cache) Put(key string, value []byte) {
	if err := c.db.Put([]byte(key), value, nil); err != nil {
		httpcache.GetLogger().Warn("failed to write to leveldb cache", slog.Any("error", err))
	}
}

// Del removes a value from the cache for the specified key. If an error occurs it is logged.
func (c *Cache) Del(key string) {
	if err := c.db.Delete([]byte(key), nil); err != nil {
		httpcache.GetLogger().Warn("failed to delete from leveldb cache", slog.Any("error", err))
	}
}

// Close closes the underlying leveldb database.
// Implements io.Closer.
func (c *Cache) Close() error {
	return c.db.Close()
}
