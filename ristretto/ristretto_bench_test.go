package ristretto_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.rtnl.ai/httpcache/ristretto"
)

func benchmarkGet(size int) func(b *testing.B) {
	return func(b *testing.B) {
		cache, err := ristretto.New(&ristretto.Config{
			NumCounters: 1e7,     // number of keys to track frequency of (10M).
			MaxCost:     1 << 30, // maximum cost of cache (1GB).
			BufferItems: 64,      // number of keys per Get buffer.
		})

		require.NoError(b, err)
		value := make([]byte, size)

		// Prepopulate the cache
		for i := 0; i < 128; i++ {
			key := string(rune('a' + i))
			cache.Put(key, value)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			cache.Get(string(rune('a' + i%192)))
		}
	}
}

func BenchmarkRistrettoCacheGet(b *testing.B) {
	b.Run("Small", benchmarkGet(512))
	b.Run("Realistic", benchmarkGet(2048))
	b.Run("Large", benchmarkGet(5.243e+6))
}

func benchmarkPut(size int) func(b *testing.B) {
	return func(b *testing.B) {
		cache, err := ristretto.New(&ristretto.Config{
			NumCounters: 1e7,     // number of keys to track frequency of (10M).
			MaxCost:     1 << 30, // maximum cost of cache (1GB).
			BufferItems: 64,      // number of keys per Get buffer.
		})

		require.NoError(b, err)
		value := make([]byte, size)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			cache.Put(string(rune('a'+i%192)), value)
		}
	}
}

func BenchmarkRistrettoCachePut(b *testing.B) {
	b.Run("Small", benchmarkPut(512))
	b.Run("Realistic", benchmarkPut(2048))
	b.Run("Large", benchmarkPut(5.243e+6))
}

// Benchmark mixed operations
func BenchmarkRistrettoCacheMixed(b *testing.B) {
	cache, err := ristretto.New(&ristretto.Config{
		NumCounters: 1e7,     // number of keys to track frequency of (10M).
		MaxCost:     1 << 30, // maximum cost of cache (1GB).
		BufferItems: 64,      // number of keys per Get buffer.
	})

	require.NoError(b, err)
	value := make([]byte, 1024)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := string(rune('a' + i%128))
		switch i % 3 {
		case 0:
			cache.Put(key, value)
		case 1:
			cache.Get(key)
		case 2:
			cache.Del(key)
		}
	}
}

// Benchmark concurrent mixed operations
func BenchmarkRistrettoCacheParallelMixed(b *testing.B) {
	cache, err := ristretto.New(&ristretto.Config{
		NumCounters: 1e7,     // number of keys to track frequency of (10M).
		MaxCost:     1 << 30, // maximum cost of cache (1GB).
		BufferItems: 64,      // number of keys per Get buffer.
	})

	require.NoError(b, err)
	value := make([]byte, 1024)

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := string(rune('a' + i%128))
			switch i % 3 {
			case 0:
				cache.Put(key, value)
			case 1:
				cache.Get(key)
			case 2:
				cache.Del(key)
			}
			i++
		}
	})
}
