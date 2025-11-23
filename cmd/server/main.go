package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"test_avito/internal/api"
	"test_avito/internal/api/handlers"
	"test_avito/internal/database"
	"test_avito/internal/repository"
	"test_avito/internal/service"
	"test_avito/pkg/config"
	"test_avito/pkg/logger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	appLogger := logger.New(cfg.Log.Level, cfg.Log.Format, os.Stdout)
	appLogger.Info("starting PR reviewer service",
		"version", "1.0.0",
		"port", cfg.Server.Port,
	)

	ctx := context.Background()
	db, err := database.New(ctx, &cfg.Database, appLogger)
	if err != nil {
		appLogger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Инициализация репозиториев
	teamRepo := repository.NewTeamRepository(db.Pool, appLogger)
	userRepo := repository.NewUserRepository(db.Pool, appLogger)
	prRepo := repository.NewPullRequestRepository(db.Pool, appLogger)
	statsRepo := repository.NewStatsRepository(db.Pool, appLogger)

	// Инициализация сервисов
	teamService := service.NewTeamService(teamRepo, userRepo, appLogger)
	userService := service.NewUserService(userRepo, appLogger)
	prService := service.NewPullRequestService(prRepo, userRepo, appLogger)
	statsService := service.NewStatsService(statsRepo, appLogger)

	// Инициализация хендлеров
	handler := handlers.NewHandler(teamService, userService, prService, statsService, appLogger)

	// Инициализация роутера и мидлваре
	router := api.NewRouter(handler, appLogger)

	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	go func() {
		appLogger.Info("server listening", "address", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			appLogger.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	appLogger.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		appLogger.Error("server forced to shutdown", "error", err)
		os.Exit(1)
	}

	appLogger.Info("server stopped gracefully")
}
