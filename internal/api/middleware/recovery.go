package middleware

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Отлавливает паники и логирует их с помощью slog
func Recovery(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				requestID := "unknown"
				if val, exists := c.Get("request_id"); exists {
					if strVal, ok := val.(string); ok {
						requestID = strVal
					}
				}

				logger.Error("panic recovered",
					slog.Any("error", err),
					slog.String("request_id", requestID),
					slog.String("path", c.Request.URL.Path),
					slog.String("method", c.Request.Method),
				)

				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": gin.H{
						"code":    "INTERNAL_ERROR",
						"message": "internal server error",
					},
				})
			}
		}()

		c.Next()
	}
}
