# HTTP Cache

**A Transport for http.Client that implements RFC 9111 - HTTP Caching for server responses.**


Inspired by [github.com/sandrolain/httpcache](https://github.com/sandrolain/httpcache), `httpcache` provides an `http.RoundTripper` implementation that works as a RFC 9111 (HTTP Caching) compliant cache for HTTP responses.