// Package logging provides structured logging helpers built on log/slog.
package logging

import (
	"log/slog"
	"os"
	"strings"
)

// New builds a slog.Logger at the given level ("debug"|"info"|"warn"|"error")
// in either "text" or "json" format.
func New(level, format string) *slog.Logger {
	opts := &slog.HandlerOptions{Level: parseLevel(level)}

	var handler slog.Handler
	if strings.EqualFold(format, "json") {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)
	return logger
}

// Default returns the process-wide default logger.
func Default() *slog.Logger { return slog.Default() }

func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
