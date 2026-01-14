package httpcache_test

import (
	"io"
	"testing"

	"github.com/stretchr/testify/require"
	"go.rtnl.ai/httpcache"
)

func TestCachingReadCloser(t *testing.T) {
	mock := &MockBody{
		data: []byte("Hello, World!"),
	}

	r := &httpcache.CachingReadCloser{
		R: mock,
		OnEOF: func(r io.Reader) {
			// Read the full content from the reader passed to OnEOF
			data, err := io.ReadAll(r)
			require.NoError(t, err)
			require.Equal(t, []byte("Hello, World!"), data, "OnEOF should receive the full content")
		},
	}

	read, err := io.ReadAll(r)
	require.NoError(t, err)
	require.Equal(t, []byte("Hello, World!"), read, "ReadAll should return the full content")

	require.NoError(t, r.Close(), "Close should not return an error")

	mock.RequireClosed(t)
	mock.RequireEmpty(t)
}

type MockBody struct {
	data     []byte
	readErr  error
	closed   bool
	closeErr error
}

func (mb *MockBody) Read(p []byte) (n int, err error) {
	if mb.readErr != nil {
		return 0, mb.readErr
	}

	if len(mb.data) == 0 {
		return 0, io.EOF
	}

	n = copy(p, mb.data)
	mb.data = mb.data[n:]
	return n, nil
}

func (mb *MockBody) Close() error {
	mb.closed = true
	return mb.closeErr
}

func (mb *MockBody) RequireClosed(t *testing.T) {
	require.True(t, mb.closed, "expected body to be closed")
}

func (mb *MockBody) RequireEmpty(t *testing.T) {
	require.Empty(t, mb.data, "expected body to be empty")
}
