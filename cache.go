package httpcache

import (
	"bufio"
	"bytes"
	"net/http"
	"sort"
	"strings"
)

// Cache implements the basic mechanism to store and retrieve responses.
type Cache interface {
	// Get returns the []byte representation of a cached response and a boolean
	// indicating whether the response was found in the cache.
	Get(string) ([]byte, bool)

	// Put stores the []byte representation of a response in the cache with a key.
	Put(string, []byte)

	// Del removes the cached response associated with the key.
	Del(string)
}

// CachedResponse returns the cached http.Response for the request if present and nil
// otherwise. Used to quickly create a client-side response from the cache.
func CachedResponse(cache Cache, req *http.Request) (rep *http.Response, err error) {
	val, ok := cache.Get(cacheKey(req))
	if !ok {
		return nil, nil
	}

	buf := bytes.NewBuffer(val)
	return http.ReadResponse(bufio.NewReader(buf), req)
}

// cachedResponse is an internal function that creates an http.Response from a cached
// value as returned by the specified key. Used internally by the Transport to handle
// headers and vary keys.
func cachedResponse(cache Cache, key string, req *http.Request) (rep *http.Response, err error) {
	val, ok := cache.Get(key)
	if !ok {
		return nil, nil
	}

	buf := bytes.NewBuffer(val)
	return http.ReadResponse(bufio.NewReader(buf), req)
}

// cacheKey returns the cache key for the given request.
func cacheKey(req *http.Request) string {
	if req.Method == http.MethodGet {
		return req.URL.String()
	}
	return req.Method + " " + req.URL.String()
}

// cacheKeyWithHeaders returns the cache key for a request and includes the specified
// headers in their canonical form. This allows you to differentiate cache entries
// based on header values such as Authorization or custom headers.
func cacheKeyWithHeaders(req *http.Request, headers []string) string {
	// Create base cachekey
	key := cacheKey(req)

	// Important - return the base key if no headers specified so that we can use this
	// method without checking the length of headers separately.
	if len(headers) == 0 {
		return key
	}

	// Append header values to the key if headers are specified
	parts := make([]string, 0, len(headers))
	for _, header := range headers {
		canonical := http.CanonicalHeaderKey(header)
		if value := normalize(req.Header.Get(canonical)); value != "" {
			parts = append(parts, canonical+":"+value)
		}
	}

	if len(parts) > 0 {
		// Sort header parts to ensure consistent ordering
		sort.Strings(parts)
		key = key + "|" + strings.Join(parts, "|")
	}

	return key
}

// cacheKeyWithVary returns the cache key for a request, including Vary headers from
// the cached response. This implements RFC 9111 vary seperation. Header values are
// normalized before inclusion in the cache key.
func cacheKeyWithVary(req *http.Request, varyHeaders []string) string {
	key := cacheKey(req)

	if len(varyHeaders) == 0 {
		return key
	}

	parts := make([]string, 0, len(varyHeaders))
	for _, header := range varyHeaders {
		canonical := http.CanonicalHeaderKey(header)
		if canonical == "" || canonical == "*" {
			continue
		}

		value := normalize(req.Header.Get(canonical))
		parts = append(parts, canonical+":"+value)
	}

	if len(parts) > 0 {
		// Sort header parts to ensure consistent ordering
		sort.Strings(parts)
		key = key + "|vary:" + strings.Join(parts, "|")
	}

	return key
}
