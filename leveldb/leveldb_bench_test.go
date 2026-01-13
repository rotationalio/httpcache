package leveldb_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.rtnl.ai/httpcache/leveldb"
)

func benchmarkGet(size int) func(b *testing.B) {
	return func(b *testing.B) {
		path := filepath.Join(b.TempDir(), "cache.db")
		cache, err := leveldb.New(path)
		require.NoError(b, err)
		defer cache.Close()

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

func BenchmarkLevelDBCacheGet(b *testing.B) {
	b.Run("Small", benchmarkGet(512))
	b.Run("Realistic", benchmarkGet(2048))
	b.Run("Large", benchmarkGet(5.243e+6))
}

func benchmarkPut(size int) func(b *testing.B) {
	return func(b *testing.B) {
		path := filepath.Join(b.TempDir(), "cache.db")
		cache, err := leveldb.New(path)
		require.NoError(b, err)
		defer cache.Close()

		value := make([]byte, size)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			cache.Put(string(rune('a'+i%192)), value)
		}
	}
}

func BenchmarkLevelDBCachePut(b *testing.B) {
	b.Run("Small", benchmarkPut(512))
	b.Run("Realistic", benchmarkPut(2048))
	b.Run("Large", benchmarkPut(5.243e+6))
}

// Benchmark mixed operations
func BenchmarkLevelDBCacheMixed(b *testing.B) {
	path := filepath.Join(b.TempDir(), "cache.db")
	cache, err := leveldb.New(path)
	require.NoError(b, err)
	defer cache.Close()

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
func BenchmarkLevelDBCacheParallelMixed(b *testing.B) {
	path := filepath.Join(b.TempDir(), "cache.db")
	cache, err := leveldb.New(path)
	require.NoError(b, err)
	defer cache.Close()

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
