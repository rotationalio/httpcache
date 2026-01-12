package ristretto

import "github.com/dgraph-io/ristretto/v2"

// Config is copied from ristretto.Config and uses the httpcache key and value types.
// It allows users to configure the Ristretto cache used by the Ristretto-backed with
// the documentation stored in httpcache rather than ristretto.
type Config struct {
	// NumCounters determines the number of counters (keys) to keep that hold
	// access frequency information. It's generally a good idea to have more
	// counters than the max cache capacity, as this will improve eviction
	// accuracy and subsequent hit ratios.
	//
	// For example, if you expect your cache to hold 1,000,000 items when full,
	// NumCounters should be 10,000,000 (10x). Each counter takes up roughly
	// 3 bytes (4 bits for each counter * 4 copies plus about a byte per
	// counter for the bloom filter). Note that the number of counters is
	// internally rounded up to the nearest power of 2, so the space usage
	// may be a little larger than 3 bytes * NumCounters.
	//
	// We've seen good performance in setting this to 10x the number of items
	// you expect to keep in the cache when full.
	NumCounters int64

	// MaxCost is how eviction decisions are made. For example, if MaxCost is
	// 100 and a new item with a cost of 1 increases total cache cost to 101,
	// 1 item will be evicted.
	//
	// MaxCost can be considered as the cache capacity, in whatever units you
	// choose to use.
	//
	// For example, if you want the cache to have a max capacity of 100MB, you
	// would set MaxCost to 100,000,000 and pass an item's number of bytes as
	// the `cost` parameter for calls to Set. If new items are accepted, the
	// eviction process will take care of making room for the new item and not
	// overflowing the MaxCost value.
	//
	// MaxCost could be anything as long as it matches how you're using the cost
	// values when calling Set.
	MaxCost int64

	// BufferItems determines the size of Get buffers.
	//
	// Unless you have a rare use case, using `64` as the BufferItems value
	// results in good performance.
	//
	// If for some reason you see Get performance decreasing with lots of
	// contention (you shouldn't), try increasing this value in increments of 64.
	// This is a fine-tuning mechanism and you probably won't have to touch this.
	BufferItems int64

	// Metrics is true when you want variety of stats about the cache.
	// There is some overhead to keeping statistics, so you should only set this
	// flag to true when testing or throughput performance isn't a major factor.
	Metrics bool

	// OnEvict is called for every eviction with the evicted item.
	OnEvict func(item *ristretto.Item[[]byte])

	// OnReject is called for every rejection done via the policy.
	OnReject func(item *ristretto.Item[[]byte])

	// OnExit is called whenever a value is removed from cache. This can be
	// used to do manual memory deallocation. Would also be called on eviction
	// as well as on rejection of the value.
	OnExit func(val []byte)

	// ShouldUpdate is called when a value already exists in cache and is being updated.
	// If ShouldUpdate returns true, the cache continues with the update (Set). If the
	// function returns false, no changes are made in the cache. If the value doesn't
	// already exist, the cache continue with setting that value for the given key.
	//
	// In this function, you can check whether the new value is valid. For example, if
	// your value has timestamp associated with it, you could check whether the new
	// value has the latest timestamp, preventing you from setting an older value.
	ShouldUpdate func(cur, prev []byte) bool

	// KeyToHash function is used to customize the key hashing algorithm.
	// Each key will be hashed using the provided function. If keyToHash value
	// is not set, the default keyToHash function is used.
	//
	// Ristretto has a variety of defaults depending on the underlying interface type
	// https://github.com/dgraph-io/ristretto/blob/main/z/z.go#L19-L41).
	//
	// Note that if you want 128bit hashes you should use the both the values
	// in the return of the function. If you want to use 64bit hashes, you can
	// just return the first uint64 and return 0 for the second uint64.
	KeyToHash func(key string) (uint64, uint64)

	// Cost evaluates a value and outputs a corresponding cost. This function is ran
	// after Set is called for a new item or an item is updated with a cost param of 0.
	//
	// Cost is an optional function you can pass to the Config in order to evaluate
	// item cost at runtime, and only when the Set call isn't going to be dropped. This
	// is useful if calculating item cost is particularly expensive and you don't want to
	// waste time on items that will be dropped anyways.
	//
	// To signal to Ristretto that you'd like to use this Cost function:
	//   1. Set the Cost field to a non-nil function.
	//   2. When calling Set for new items or item updates, use a `cost` of 0.
	Cost func(value []byte) int64

	// IgnoreInternalCost set to true indicates to the cache that the cost of
	// internally storing the value should be ignored. This is useful when the
	// cost passed to set is not using bytes as units. Keep in mind that setting
	// this to true will increase the memory usage.
	IgnoreInternalCost bool

	// TtlTickerDurationInSec sets the value of time ticker for cleanup keys on TTL expiry.
	TtlTickerDurationInSec int64
}

func (c *Config) convert() *ristretto.Config[string, []byte] {
	return &ristretto.Config[string, []byte]{
		NumCounters:            c.NumCounters,
		MaxCost:                c.MaxCost,
		BufferItems:            c.BufferItems,
		Metrics:                c.Metrics,
		OnEvict:                c.OnEvict,
		OnReject:               c.OnReject,
		OnExit:                 c.OnExit,
		ShouldUpdate:           c.ShouldUpdate,
		KeyToHash:              c.KeyToHash,
		Cost:                   c.Cost,
		IgnoreInternalCost:     c.IgnoreInternalCost,
		TtlTickerDurationInSec: c.TtlTickerDurationInSec,
	}
}
