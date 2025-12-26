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
