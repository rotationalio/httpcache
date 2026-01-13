package httpcache_test

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"go.rtnl.ai/httpcache"
)

//===========================================================================
// Testing Helpers
//===========================================================================s

type TestRequest struct {
	method  string
	url     string
	body    []byte
	headers map[string]string
}

func (tr *TestRequest) HTTP() *http.Request {
	if tr.method == "" {
		tr.method = http.MethodGet
	}

	var body io.Reader
	if len(tr.body) > 0 {
		body = bytes.NewReader(tr.body)
	}

	req, _ := http.NewRequest(tr.method, tr.url, body)
	for k, v := range tr.headers {
		req.Header.Set(k, v)
	}

	return req
}

//===========================================================================
// Package Helpers Testing
//===========================================================================

func TestNormalize(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"  Hello   World  ", "Hello World"},                      // Leading/trailing and multiple spaces
		{"Line1\nLine2\r\nLine3", "Line1 Line2 Line3"},            // Newlines and carriage returns
		{"Value1, Value2,Value3", "Value1,Value2,Value3"},         // Comma-separated values
		{"\tTabbed\tText\t", "Tabbed Text"},                       // Tabs
		{" Value1,\tValue2,\t\t\tValue3", "Value1,Value2,Value3"}, // Mixed whitespace in comma-separated values
		{"Single Value", "Single Value"},                          // No normalization needed
		{"", ""},                                                  // Empty string
	}

	for _, test := range tests {
		result := httpcache.Normalize(test.input)
		require.Equal(t, test.expected, result)
	}
}

func TestAllHeaderCSVs(t *testing.T) {
	tests := []struct {
		name     string
		headers  http.Header
		header   string
		expected []string
	}{
		{
			name: "Single header with single value",
			headers: http.Header{
				"Accept": []string{"text/html"},
			},
			header:   "Accept",
			expected: []string{"text/html"},
		},
		{
			name: "Single header with single CSV",
			headers: http.Header{
				"Accept": []string{"text/html, application/json"},
			},
			header:   "Accept",
			expected: []string{"text/html", "application/json"},
		},
		{
			name: "Multiple headers with single values",
			headers: http.Header{
				"Accept": []string{"text/html", "application/xml"},
			},
			header:   "Accept",
			expected: []string{"text/html", "application/xml"},
		},
		{
			name: "Multiple Headers with multiple CSVs",
			headers: http.Header{
				"Accept": []string{"text/html, application/json", "application/xml, text/plain, application/yaml"},
			},
			header:   "Accept",
			expected: []string{"text/html", "application/json", "application/xml", "text/plain", "application/yaml"},
		},
		{
			name: "Header not present",
			headers: http.Header{
				"Accept-Language": []string{"en,fr,de"},
			},
			header:   "Accept",
			expected: nil,
		},
		{
			name: "Canonicalize header name",
			headers: http.Header{
				"Accept-Language": []string{"en,fr,de"},
			},
			header:   "accept-language",
			expected: []string{"en", "fr", "de"},
		},
		{
			name: "Trim spaces in CSV values",
			headers: http.Header{
				"Accept-Language": []string{"en,fr,      de    , es "},
			},
			header:   "accept-language",
			expected: []string{"en", "fr", "de", "es"},
		},
		{
			name: "Case Sensitive CSV values",
			headers: http.Header{
				"Accept-Language": []string{"en-US,fr-AG,es-MX"},
			},
			header:   "accept-language",
			expected: []string{"en-US", "fr-AG", "es-MX"},
		},
	}

	for _, test := range tests {
		result := httpcache.AllHeaderCSVs(test.headers, test.header)
		require.Equal(t, test.expected, result, "Failed test: %s", test.name)
	}
}

func TestIsUnsafeMethod(t *testing.T) {
	tests := []struct {
		method  string
		require require.BoolAssertionFunc
	}{
		{"GET", require.False},
		{"HEAD", require.False},
		{"OPTIONS", require.False},
		{"POST", require.True},
		{"PUT", require.True},
		{"DELETE", require.True},
		{"PATCH", require.True},
		{"TRACE", require.False},
	}

	for _, test := range tests {
		result := httpcache.IsUnsafeMethod(test.method)
		test.require(t, result, "Failed for method: %q", test.method)
	}
}
