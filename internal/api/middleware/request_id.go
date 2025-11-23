package middleware

import (
	"test_avito/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RequestID добавляем уникальный идентификатор запроса к каждому запрос
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		c.Set(string(logger.RequestIDKey), requestID)

		c.Header("X-Request-ID", requestID)

		c.Next()
	}
}
