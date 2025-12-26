package httpcache_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.rtnl.ai/httpcache"
)

func TestCacheKey(t *testing.T) {
	tests := []struct {
		name     string
		request  *TestRequest
		expected string
	}{
		{
			name:     "Simple GET Request",
			request:  &TestRequest{method: "GET", url: "http://example.com/resource"},
			expected: "http://example.com/resource",
		},
		{
			name:     "Simple POST Request",
			request:  &TestRequest{method: "POST", url: "http://example.com/resource"},
			expected: "POST http://example.com/resource",
		},
		{
			name:     "GET Request with Query Params",
			request:  &TestRequest{method: "GET", url: "https://example.com/resource?id=123"},
			expected: "https://example.com/resource?id=123",
		},
		{
			name:     "PUT Request with Query Params",
			request:  &TestRequest{method: "PUT", url: "https://example.com/resource?id=123"},
			expected: "PUT https://example.com/resource?id=123",
		},
	}

	for _, test := range tests {
		result := httpcache.CacheKey(test.request.HTTP())
		require.Equal(t, test.expected, result, "Test Case: %q", test.name)
	}
}

func TestCacheKeyWithHeaders(t *testing.T) {
	tests := []struct {
		name     string
		request  *TestRequest
		headers  []string
		expected string
	}{
		{
			name: "No headers (nil)",
			request: &TestRequest{
				method:  "GET",
				url:     "http://example.com/resource",
				headers: map[string]string{"Accept": "   application/json  ", "Accept-Language": "en-US, fr"}},
			headers:  nil,
			expected: "http://example.com/resource",
		},
		{
			name: "No headers (empty)",
			request: &TestRequest{
				method:  "GET",
				url:     "http://example.com/resource",
				headers: map[string]string{"Accept": "   application/json  ", "Accept-Language": "en-US, fr"}},
			headers:  []string{},
			expected: "http://example.com/resource",
		},
		{
			name: "With headers, unnormalized values",
			request: &TestRequest{
				method:  "GET",
				url:     "http://example.com/resource",
				headers: map[string]string{"Accept": "   application/json  ", "Accept-Language": "en-US, fr"}},
			headers:  []string{"Accept", "Accept-Language"},
			expected: "http://example.com/resource|Accept-Language:en-US,fr|Accept:application/json",
		},
		{
			name: "With headers, request missing headers",
			request: &TestRequest{
				method:  "GET",
				url:     "http://example.com/resource",
				headers: map[string]string{"Accept-Language": "en,fr"}},
			headers:  []string{"Accept", "Accept-Language", "Authorization"},
			expected: "http://example.com/resource|Accept-Language:en,fr",
		},
		{
			name: "With headers, not canonicalized",
			request: &TestRequest{
				method:  "GET",
				url:     "http://example.com/resource",
				headers: map[string]string{"accept-language": "en,fr"}},
			headers:  []string{"accept-language"},
			expected: "http://example.com/resource|Accept-Language:en,fr",
		},
	}

	for _, test := range tests {
		result := httpcache.CacheKeyWithHeaders(test.request.HTTP(), test.headers)
		require.Equal(t, test.expected, result, "Test Case: %q", test.name)
	}
}

func TestCacheKeyWithVary(t *testing.T) {
	tests := []struct {
		name     string
		request  *TestRequest
		headers  []string
		expected string
	}{
		{
			name: "No Vary headers (nil)",
			request: &TestRequest{
				method:  "GET",
				url:     "http://example.com/resource",
				headers: map[string]string{"Accept": "text/html", "Accept-Language": "en, fr"}},
			headers:  nil,
			expected: "http://example.com/resource",
		},
		{
			name: "No Vary headers (empty)",
			request: &TestRequest{
				method:  "GET",
				url:     "http://example.com/resource",
				headers: map[string]string{"Accept": "text/html", "Accept-Language": "en, fr"}},
			headers:  []string{},
			expected: "http://example.com/resource",
		},
		{
			name: "With Vary headers, unnormalized values",
			request: &TestRequest{
				method:  "GET",
				url:     "http://example.com/resource",
				headers: map[string]string{"Accept": "   text/html  ", "Accept-Language": "en-US, fr"}},
			headers:  []string{"Accept", "Accept-Language"},
			expected: "http://example.com/resource|vary:Accept-Language:en-US,fr|Accept:text/html",
		},
		{
			name: "With Vary headers, request missing headers",
			request: &TestRequest{
				method:  "GET",
				url:     "http://example.com/resource",
				headers: map[string]string{"Accept-Language": "en,fr"}},
			headers:  []string{"Accept", "Accept-Language", "Authorization"},
			expected: "http://example.com/resource|vary:Accept-Language:en,fr|Accept:|Authorization:",
		},
		{
			name: "With Vary headers, not canonicalized",
			request: &TestRequest{
				method:  "GET",
				url:     "http://example.com/resource",
				headers: map[string]string{"accept-language": "en,fr"}},
			headers:  []string{"accept-language"},
			expected: "http://example.com/resource|vary:Accept-Language:en,fr",
		},
		{
			name: "With Vary headers, wildcard ignored",
			request: &TestRequest{
				method:  "GET",
				url:     "http://example.com/resource",
				headers: map[string]string{"Accept": "text/html", "Accept-Language": "en,fr"}},
			headers:  []string{"*", "Accept", "Accept-Language"},
			expected: "http://example.com/resource|vary:Accept-Language:en,fr|Accept:text/html",
		},
		{
			name: "With Vary headers, empty header name ignored",
			request: &TestRequest{
				method:  "GET",
				url:     "http://example.com/resource",
				headers: map[string]string{"Accept": "text/html", "Accept-Language": "en,fr"}},
			headers:  []string{"", "Accept", "Accept-Language"},
			expected: "http://example.com/resource|vary:Accept-Language:en,fr|Accept:text/html",
		},
	}

	for _, test := range tests {
		result := httpcache.CacheKeyWithVary(test.request.HTTP(), test.headers)
		require.Equal(t, test.expected, result, "Test Case: %q", test.name)
	}
}
