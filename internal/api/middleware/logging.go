package middleware

import (
	"log/slog"
	"time"

	"test_avito/pkg/logger"

	"github.com/gin-gonic/gin"
)

// Создает middleware для структурированного логирования с помощью slog
func Logging(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		requestID := "unknown"
		if val, exists := c.Get(string(logger.RequestIDKey)); exists {
			if strVal, ok := val.(string); ok {
				requestID = strVal
			}
		}

		// Создаем логгер с контекстом запроса
		reqLogger := log.With(
			slog.String("request_id", requestID),
			slog.String("method", c.Request.Method),
			slog.String("path", path),
			slog.String("ip", c.ClientIP()),
		)

		// Добавляем логгер в контекст Gin для хэндлеров
		c.Set("logger", reqLogger)

		c.Next()

		latency := time.Since(start)
		statusCode := c.Writer.Status()

		// Создаем поля для логирования
		logFields := []interface{}{
			slog.Int("status", statusCode),
			slog.Duration("latency", latency),
			slog.Int("body_size", c.Writer.Size()),
		}

		if raw != "" {
			logFields = append(logFields, slog.String("query", raw))
		}

		if len(c.Errors) > 0 {
			logFields = append(logFields, slog.String("error", c.Errors.String()))
		}

		// Логируем в зависимости от статуса ответа
		switch {
		case statusCode >= 500:
			reqLogger.Error("server error", logFields...)
		case statusCode >= 400:
			reqLogger.Warn("client error", logFields...)
		default:
			reqLogger.Info("request completed", logFields...)
		}
	}
}
