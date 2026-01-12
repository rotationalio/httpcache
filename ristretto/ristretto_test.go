package ristretto_test

import (
	"math/rand/v2"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"go.rtnl.ai/httpcache/ristretto"
)

func TestRistrettoCache(t *testing.T) {
	cache, err := ristretto.New(&ristretto.Config{
		NumCounters: 1e7,     // number of keys to track frequency of (10M).
		MaxCost:     1 << 30, // maximum cost of cache (1GB).
		BufferItems: 64,      // number of keys per Get buffer.
	})
	require.NoError(t, err)

	cache.Put("foo", []byte("bar"))
	cache.Wait()

	val, ok := cache.Get("foo")
	require.True(t, ok)
	require.Equal(t, []byte("bar"), val)

	cache.Del("foo")
	_, ok = cache.Get("foo")
	require.False(t, ok)
}

func TestRistrettoRace(t *testing.T) {
	// Ensures no race conditions occur during concurrent access.
	cache, err := ristretto.New(&ristretto.Config{
		NumCounters: 1e7,     // number of keys to track frequency of (10M).
		MaxCost:     1 << 30, // maximum cost of cache (1GB).
		BufferItems: 64,      // number of keys per Get buffer.
	})
	require.NoError(t, err)
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
