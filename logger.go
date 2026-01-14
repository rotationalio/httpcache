package httpcache

import (
	"log/slog"
	"sync"
)

var (
	logger   *slog.Logger
	loggerMu sync.Once
)

// SetLogger sets a custom slog.Logger instance to be used by httpcache. If not set,
// the default slog logger will be used. Rotational apps should implement a zerolog
// slogger for observability.
//
// To stop logging either pass slog.New(slog.DiscardHandler) or nil (which will
// create a discard handler).
func SetLogger(l *slog.Logger) {
	if l != nil {
		logger = l
		return
	}

	// Set a discard handler if nil is provided.
	logger = slog.New(slog.DiscardHandler)
}

// GetLogger returns the configured logger or the default slog logger.
func GetLogger() *slog.Logger {
	loggerMu.Do(func() {
		if logger == nil {
			logger = slog.Default()
		}
	})
	return logger
}
