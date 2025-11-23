package api

import (
	"log/slog"

	"test_avito/internal/api/handlers"
	"test_avito/internal/api/middleware"

	"github.com/gin-gonic/gin"
)

func NewRouter(handler *handlers.Handler, logger *slog.Logger) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()

	r.Use(middleware.RequestID())
	r.Use(middleware.Recovery(logger))
	r.Use(middleware.Logging(logger))

	handler.RegisterRoutes(r)

	return r
}
