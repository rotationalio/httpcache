package leveldb_test

import (
	"math/rand/v2"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"go.rtnl.ai/httpcache/leveldb"
)

func TestLevelDBCache(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cache.db")

	cache, err := leveldb.New(path)
	require.NoError(t, err)
	defer cache.Close()

	cache.Put("foo", []byte("bar"))

	val, ok := cache.Get("foo")
	require.True(t, ok)
	require.Equal(t, []byte("bar"), val)

	cache.Del("foo")
	_, ok = cache.Get("foo")
	require.False(t, ok)
}

func TestLevelDBRace(t *testing.T) {
	// Ensures no race conditions occur during concurrent access.
	path := filepath.Join(t.TempDir(), "cache.db")
	cache, err := leveldb.New(path)
	require.NoError(t, err)
	defer cache.Close()

	value := make([]byte, 2048)

	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 512; j++ {
				k := rand.IntN(64)
				key := string(rune('a' + k%16))
				switch k % 3 {
				case 0:
					cache.Put(key, value)
				case 1:
					cache.Get(key)
				case 2:
					cache.Del(key)
				}
			}
		}()
	}
	wg.Wait()
}
