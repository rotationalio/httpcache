package httpcache

import (
	"net/http"
	"strings"
	"time"
)

//===========================================================================
// Transport
//===========================================================================

// Transport is an implementation of http.RoundTripper that will return values from a
// cache here possible (avoiding a network request) and will additionally add
// request cache headers (etag/if-modified-since) to repeated requests allowing servers
// to return cache-oriented responses such as 304 Not Modified.
type Transport struct {
	Transport http.RoundTripper
	Cache     Cache

	// If true, responses returned from the cache will include an X-From-Cache header.
	MarkCachedResponses bool

	// If true, server errors (5xx status codes) will be served from the cache. By
	// default, this is false, forcing a new request on server errors.
	ServerErrorsFromCache bool

	// Timeout for async requests triggered by stale-while-revalidate responses.
	// If zero, no timeout is applied to async revalidation requests.
	AsyncRevalidateTimeout time.Duration

	// IsPublicCache enables public cache mode (default is false for private cache).
	// When true, the cache will NOT store responses with the Cache-Control: private
	// directive. When false (default), the cache CAN store private responses.
	// Set to true only if using httpcache as a shared/public cache (e.g. a reverse
	// proxy server or CDN).
	IsPublicCache bool

	// Disables RFC 9111 compliant Vary header separation (default: false).
	// When true, responses with Vary headers are ignored and each variant overwrites
	// the previous one in the cache. This is useful to reduce cache storage space since
	// by default each variant is stored separately based on the Vary header values.
	DisableVaryHeaderSeparation bool

	// Used to configure non-standard caching behavior based on the response. If set,
	// this function is used to determine whether a non-200 response should be cached.
	// This enables caching of responses like 404 Not Found or other status codes. If
	// nil, only 200 OK responses are cached by default.
	//
	// The function receives the http.Response and should return true to cache it.
	// NOTE: this only bypasses the status code check; Cache-Control headers are still
	// respected.
	ShouldCache func(resp *http.Response) bool

	// Specify additional request headers to include in the cache key generation
	// allowing storage of separate cache entries based on these headers. For example,
	// specifying the Authorization header allows user-specific caching or the
	// Accept-Language header allows caching based on language preferences.
	// Header names are case-insensitive and will be converted to their canonical form.
	// NOTE: this is different from the Vary header handling which is server-driven.
	CacheKeyHeaders []string
}

// NewTransport returns a new Transport with provided Cache implementation and
// MarkCachedResponses set to true (default httpcache behavior).
func NewTransport(cache Cache) *Transport {
	return &Transport{
		Cache:               cache,
		MarkCachedResponses: true,
	}
}

// Client returns a new http.Client that caches respones.
func (t *Transport) Client() *http.Client {
	return &http.Client{Transport: t}
}

// Execute an http Request and return an http Response.
//
// If there is a fresh Response already in the cache, it will be returned without
// connecting to the server.
//
// If there is a stale Response in the cache, then any validators it contains will be
// set on the new request to give the server a chance to respond with 304 Not Modified.
// In this case, the cached Response will be returned.
//
// RoundTrip implements the http.RoundTripper interface for Transport.
func (t *Transport) RoundTrip(req *http.Request) (rep *http.Response, err error) {
	cacheKey := cacheKeyWithHeaders(req, t.CacheKeyHeaders)
	cacheable := (req.Method == http.MethodGet || req.Method == http.MethodHead) && req.Header.Get("Range") == ""

	var cached *http.Response
	if cacheable {
		// Attempt to get a cached response
		cached, err = cachedResponse(t.Cache, cacheKey, req)

		// RFC 9111 Vary Separation: if not disabled and cached response has Vary
		// headers, recalculate the cache key with vary values and try again for the
		// correct variant.
		if !t.DisableVaryHeaderSeparation && cached != nil && err == nil {
			if varyHeaders := allHeaderCSVs(cached.Header, "Vary"); len(varyHeaders) > 0 {
				varyKey := cacheKeyWithVary(req, varyHeaders)
				if varyKey != cacheKey {
					// Try with vary-specific key
					cachedVery, varyErr := cachedResponse(t.Cache, varyKey, req)
					if varyErr == nil && cachedVery != nil {
						// Found a variant match
						cached = cachedVery
						cacheKey = varyKey
					}
				}
			}
		}

	} else {
		// RFC 7234 Section 4.4: Invalidate cache on unsafe methods
		// Delete the request URI immediately for unsafe methods
		t.Cache.Del(cacheKey)
	}

	// Use default http transport if not specified
	transport := t.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}

	// Handle cached vs uncached response
	if cacheable && cached != nil && err == nil {
		rep, err = t.processCachedResponse(cached, req, transport, cacheKey)
	} else {
		rep, err = t.processUncachedRequest(transport, req)
	}

	// Error handling for multiple error phases above.
	if err != nil {
		return nil, err
	}

	// RFC 7234 Section 4.4: Invalidate cache for unsafe methods
	// After successful response, invalidate related URIs
	if isUnsafeMethod(req.Method) {
		t.invalidateCache(req, rep)
	}

	// Store response in cache if applicable
	t.storeResponseInCache(req, rep, cacheKey, cacheable)
	return rep, nil
}

func (t *Transport) processCachedResponse(cached *http.Response, req *http.Request, transport http.RoundTripper, cacheKey string) (rep *http.Response, err error) {
	return nil, nil
}

func (t *Transport) processUncachedRequest(transport http.RoundTripper, req *http.Request) (rep *http.Response, err error) {
	return transport.RoundTrip(req)
}

func (t *Transport) invalidateCache(req *http.Request, resp *http.Response) {}

func (t *Transport) storeResponseInCache(req *http.Request, resp *http.Response, cacheKey string, cacheable bool) {
}

//===========================================================================
// Top Level Helper Functions
//===========================================================================

const (
	nbsp = ' '
)

func normalize(value string) string {
	// Trim leading/trailing whitespace
	value = strings.TrimSpace(value)

	// Normalize all whitespace sequences to a single space
	var (
		norm      strings.Builder
		prevSpace bool
	)

	for _, c := range value {
		if c == nbsp || c == '\t' || c == '\n' || c == '\r' {
			if !prevSpace {
				norm.WriteRune(nbsp)
				prevSpace = true
			}
		} else {
			norm.WriteRune(c)
			prevSpace = false
		}
	}

	// Normalize comma-separated values (e.g. en,fr and en, fr should match)
	result := strings.ReplaceAll(norm.String(), ", ", ",")
	return result
}

func allHeaderCSVs(headers http.Header, header string) []string {
	var vals []string
	for _, val := range headers[http.CanonicalHeaderKey(header)] {
		fields := strings.Split(val, ",")
		for i, field := range fields {
			fields[i] = strings.TrimSpace(field)
		}
		vals = append(vals, fields...)
	}
	return vals
}

// isUnsafeMethod returns true if the HTTP method is considered unsafe.
// RFC 7234 Section 4.4: POST, PUT, DELETE, PATCH are unsafe methods.
func isUnsafeMethod(method string) bool {
	return method == http.MethodPost ||
		method == http.MethodPut ||
		method == http.MethodDelete ||
		method == http.MethodPatch
}
