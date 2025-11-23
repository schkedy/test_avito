// Используем slog для структурированного сквозного логирования с контекстом
package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
)

// ContextKey тип для ключей контекста
type ContextKey string

const (
	// RequestIDKey ключ в контексте для request ID
	RequestIDKey ContextKey = "request_id"
)

func New(level, format string, output io.Writer) *slog.Logger {
	if output == nil {
		output = os.Stdout
	}

	var logLevel slog.Level
	switch strings.ToLower(level) {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn", "warning":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level:     logLevel,
		AddSource: true,
	}

	var handler slog.Handler
	switch strings.ToLower(format) {
	case "json":
		handler = slog.NewJSONHandler(output, opts)
	case "text":
		handler = slog.NewTextHandler(output, opts)
	default:
		handler = slog.NewJSONHandler(output, opts)
	}

	return slog.New(handler)
}

// Добавляем request ID в логгер из контекста
func WithRequestID(ctx context.Context, logger *slog.Logger) *slog.Logger {
	if requestID, ok := ctx.Value(RequestIDKey).(string); ok {
		return logger.With(slog.String("request_id", requestID))
	}
	return logger
}

// Достаём из контекста логгер
func FromContext(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(loggerKey).(*slog.Logger); ok {
		return logger
	}
	return slog.Default()
}

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const loggerKey contextKey = "logger"

// Добавляем логгер в контекст
func ToContext(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}
