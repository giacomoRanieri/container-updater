package logger

import (
	"context"
	"log/slog"
	"os"
)

var Log *slog.Logger

func init() {
	Init("info")
}

func Init(level string) {
	var slogLevel slog.Level
	switch level {
	case "debug":
		slogLevel = slog.LevelDebug
	case "warn":
		slogLevel = slog.LevelWarn
	case "error":
		slogLevel = slog.LevelError
	default:
		slogLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: slogLevel,
	}

	// Use JSON handler for structured logs in production-like environments,
	// or text handler for local dev. Let's default to JSON handler for structured logging.
	handler := slog.NewJSONHandler(os.Stdout, opts)
	Log = slog.New(handler)
	slog.SetDefault(Log)
}

// Global error helper
func Err(ctx context.Context, msg string, err error, args ...any) {
	if Log == nil {
		Init("info")
	}
	allArgs := append([]any{"error", err}, args...)
	Log.ErrorContext(ctx, msg, allArgs...)
}
