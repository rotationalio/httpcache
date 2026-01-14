package httpcache

import (
	"bytes"
	"io"
)

// CachingReadCloser wraps a ReadCloser R that calls the OnEOF handler with a full copy
// of the content read from R when EOF is reached. While this does cause the full read
// to be stored in memory, it allows the Transport to cache the response body only once
// it has been fully read.
type CachingReadCloser struct {
	// Wrapped ReadCloser (response.Body)
	R io.ReadCloser

	// Called when EOF is reached with a reader containing the full content read.
	OnEOF func(r io.Reader)

	// Internal buffer to store a copy of the content of R.
	buf bytes.Buffer
}

func (r *CachingReadCloser) Read(p []byte) (n int, err error) {
	n, err = r.R.Read(p)
	r.buf.Write(p[:n])

	if err == io.EOF || n < len(p) {
		if r.OnEOF != nil {
			r.OnEOF(bytes.NewReader(r.buf.Bytes()))
		}
	}

	return n, err
}

func (r *CachingReadCloser) Close() error {
	return r.R.Close()
}
