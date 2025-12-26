package httpcache

import "net/http"

type Cache interface{}

type Transport struct {
	Transport http.RoundTripper
}
