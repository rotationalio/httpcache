package httpcache

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	"go.rtnl.ai/x/httpcc"
)

// RFC 9111 Section 5.2.2.3: HTTP status codes that are understood by this cache.
// When must-understand directive is present, only responses with these status codes
// can be cached, even if other cache directives would normally prevent caching.
var understoodStatusCodes = map[int]bool{
	http.StatusOK:                   true, // 200
	http.StatusNonAuthoritativeInfo: true, // 203
	http.StatusNoContent:            true, // 204
	http.StatusPartialContent:       true, // 206
	http.StatusMultipleChoices:      true, // 300
	http.StatusMovedPermanently:     true, // 301
	http.StatusNotFound:             true, // 404
	http.StatusMethodNotAllowed:     true, // 405
	http.StatusGone:                 true, // 410
	http.StatusRequestURITooLong:    true, // 414
	http.StatusNotImplemented:       true, // 501
}

const (
	Vary          = "Vary"
	Range         = "Range"
	Authorization = "Authorization"
	XVariedPrefix = "X-Varied-"
)

//===========================================================================
// Transport
//===========================================================================

// Transport is an implementation of http.RoundTripper that will return values from a
// cache where possible (avoiding a network request) and will additionally add
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

// Client returns a new http.Client that caches responses.
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
	cacheable := (req.Method == http.MethodGet || req.Method == http.MethodHead) && req.Header.Get(Range) == ""

	var cached *http.Response
	if cacheable {
		// Attempt to get a cached response
		cached, err = cachedResponse(t.Cache, cacheKey, req)

		// RFC 9111 Vary Separation: if not disabled and cached response has Vary
		// headers, recalculate the cache key with vary values and try again for the
		// correct variant.
		if !t.DisableVaryHeaderSeparation && cached != nil && err == nil {
			if varyHeaders := allHeaderCSVs(cached.Header, Vary); len(varyHeaders) > 0 {
				varyKey := cacheKeyWithVary(req, varyHeaders)
				if varyKey != cacheKey {
					// Try with vary-specific key
					cachedVary, varyErr := cachedResponse(t.Cache, varyKey, req)
					if varyErr == nil && cachedVary != nil {
						// Found a variant match use it as the cached value.
						cached = cachedVary
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
	t.cacheResponse(req, rep, cacheKey, cacheable)
	return rep, nil
}

func (t *Transport) processCachedResponse(cached *http.Response, req *http.Request, transport http.RoundTripper, cacheKey string) (rep *http.Response, err error) {
	return nil, nil
}

func (t *Transport) processUncachedRequest(transport http.RoundTripper, req *http.Request) (rep *http.Response, err error) {
	return transport.RoundTrip(req)
}

func (t *Transport) invalidateCache(req *http.Request, rep *http.Response) {}

func (t *Transport) cacheResponse(req *http.Request, rep *http.Response, cacheKey string, cacheable bool) {
	var (
		err   error
		repcc *httpcc.ResponseDirective
		reqcc *httpcc.RequestDirective
	)

	if repcc, err = httpcc.Response(rep); err != nil {
		GetLogger().Warn("could not parse response cache-control directives", slog.Any("error", err))
		repcc = &httpcc.ResponseDirective{}
	}

	if reqcc, err = httpcc.Request(req); err != nil {
		GetLogger().Warn("could not parse request cache-control directives", slog.Any("error", err))
		reqcc = &httpcc.RequestDirective{}
	}

	if !cacheable || !t.canStore(rep, req, reqcc, repcc) {
		t.Cache.Del(cacheKey)
		return
	}

	// RFC 9111 Section 5.2.2.3: must-understand directive
	// When must-understand is present and status code is understood, always cache
	mustUnderstandAllowsCaching := repcc.MustUnderstand() && understoodStatusCodes[rep.StatusCode]
	shouldCache := understoodStatusCodes[rep.StatusCode] || mustUnderstandAllowsCaching

	// Allow custom override via ShouldCache user supplied hook.
	if !shouldCache && t.ShouldCache != nil {
		shouldCache = t.ShouldCache(rep)
	}

	if !shouldCache {
		t.Cache.Del(cacheKey)
		return
	}

	// Store all Vary headers in the response for future cache validation.
	storeVaryHeaders(rep, req)

	// Determine the cache key(s) to use.
	cacheKeys := []string{cacheKey}

	// Handle Vary headers to store the body across multiple keys.
	varyHeaders := allHeaderCSVs(rep.Header, Vary)
	if !t.DisableVaryHeaderSeparation && len(varyHeaders) > 0 {
		// Store the full response ounder both the variant key and the base key
		// so that RoundTrip can read the base entry (to discover Vary) and then
		// re-lookup the variant-specific entry.
		if varyKey := cacheKeyWithVary(req, varyHeaders); varyKey != cacheKey {
			cacheKeys = append(cacheKeys, varyKey)
		}
	}

	// Finally store the response in the cache!
	// If the request is a GET request, add a CachingReadCloser to the response body
	// so that the response is only stored once the body has been fully read; otherwise
	// store the response immediately.
	//
	// NOTE: this means that not GET requests will not be able to read the body!
	// If this is required, we need to add a setting to the Transport to handle this.
	if req.Method == http.MethodGet {
		// Setup caching on body read completion
		t.setupCachingBody(rep, cacheKeys...)
	} else {
		// Store immediately for non-GET requests
		t.store(rep, cacheKeys...)
	}
}

// Stores a response in the cache dumping it as bytes.
// NOTE: expects cache headers to already be set on the response.
func (t *Transport) store(rep *http.Response, cacheKeys ...string) {
	if t.Cache == nil {
		GetLogger().Error("cannot store response in cache: no cache configured")
		return
	}

	// Read the response body only once and store it in the cache for all provided keys.
	if data, err := httputil.DumpResponse(rep, true); err == nil {
		for _, cacheKey := range cacheKeys {
			t.Cache.Put(cacheKey, data)
		}
	} else {
		GetLogger().Error("could not dump response for caching", slog.Any("error", err))
	}
}

// Wraps the response body to be cached when it is fully read.
func (t *Transport) setupCachingBody(rep *http.Response, cacheKeys ...string) {
	rep.Body = &CachingReadCloser{
		R: rep.Body,
		OnEOF: func(r io.Reader) {
			// Convert the response body to the cached response by copying the reply
			rep := *rep
			rep.Body = io.NopCloser(r)

			// Store the response in the cache
			t.store(&rep, cacheKeys...)
		},
	}
}

// Determines if a response can be stored in the cache based on request/response
// cache-control directives and status code. This method implements RFC 911 compliance.
// RFC 9111 Section 3: Storing Responses in Caches
// RFC 9111 Section 5.2.2.3: must-understand directive
// RFC 9111 Section 3.5: Storing Responses to Authenticated Requests
func (t *Transport) canStore(rep *http.Response, req *http.Request, reqcc *httpcc.RequestDirective, repcc *httpcc.ResponseDirective) bool {
	// RFC 9111 Section 5.2.2.3: must-understand directive
	// When must-understand is present, the cache can only store the response if:
	// 1. The status code is understood by the cache, AND
	// 2. All other cache directives are comprehended
	//
	// If must-understand is present and the status code is not understood,
	// the cache MUST NOT store the response, even if other directives would permit it.
	//
	// If must-understand is present and the status code IS understood,
	// then no-store is effectively ignored (the response can be cached).
	if repcc.MustUnderstand() {
		if !understoodStatusCodes[rep.StatusCode] {
			// Status code not understood → must not cache.
			return false
		}
		// Status code understood → proceed (overrides no-store).
	} else {
		// Normal behavior when must-understand is not present.
		if repcc.NoStore() || reqcc.NoStore() {
			return false
		}
	}

	// RFC 9111 Section 3.5: Storing Responses to Authenticated Requests
	// A shared cache MUST NOT use a cached response to a request with an Authorization
	// header field unless the response contains a Cache-Control field with the "public",
	// "must-revalidate", or "s-maxage" response directive.
	if t.IsPublicCache && req.Header.Get(Authorization) != "" {
		_, hasSMaxAge := repcc.SMaxAge()
		if !repcc.Public() && !repcc.MustRevalidate() && !hasSMaxAge {
			GetLogger().Debug(
				"refusing to cache Authorization request in shared cache",
				slog.String("url", req.URL.String()),
				slog.String("reason", "no public/must-revalidate/s-maxage directive"),
				slog.Bool("public", repcc.Public()),
				slog.Bool("must-revalidate", repcc.MustRevalidate()),
				slog.Bool("has-s-maxage", hasSMaxAge),
			)
			return false
		}
	}

	// RFC 9111: Check Cache-Control: private directive
	if repcc.Private() && t.IsPublicCache {
		return false
	}

	return true
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
		for _, field := range strings.Split(val, ",") {
			if s := strings.TrimSpace(field); s != "" {
				vals = append(vals, s)
			}
		}
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

// Stores the Vary header values in the response for future cache validation.
// RFC 9111 Section 4.1: Values are normalized before storage to enable proper matching.
func storeVaryHeaders(rep *http.Response, req *http.Request) {
	for _, varyKey := range allHeaderCSVs(rep.Header, Vary) {
		varyKey := http.CanonicalHeaderKey(strings.TrimSpace(varyKey))
		if varyKey == "" || varyKey == "*" {
			continue
		}

		value := normalize(req.Header.Get(varyKey))
		cacheHeader := XVariedPrefix + varyKey
		rep.Header.Set(cacheHeader, value)
	}
}
