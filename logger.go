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
func SetLogger(l *slog.Logger) {
	logger = l
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
