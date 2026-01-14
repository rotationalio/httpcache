package httpcache

// Exports private functions but only when testing so they are not part of the public
// API but tests can be run in the httpcache_test package (works because of the _test.go
// suffix appended to this filename).
var (
	CacheKey              = cacheKey
	CacheKeyWithHeaders   = cacheKeyWithHeaders
	CacheKeyWithVary      = cacheKeyWithVary
	Normalize             = normalize
	CachedResponseWithKey = cachedResponse
	AllHeaderCSVs         = allHeaderCSVs
	IsUnsafeMethod        = isUnsafeMethod
	IsSameOrigin          = isSameOrigin
	GetOrigin             = getOrigin
)
