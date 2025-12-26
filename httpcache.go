package httpcache

import (
	"net/http"
	"strings"
)

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

//===========================================================================
// Transport
//===========================================================================

type Transport struct {
	Transport http.RoundTripper
}
