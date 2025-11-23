package database

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"test_avito/pkg/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DB struct {
	Pool   *pgxpool.Pool
	logger *slog.Logger
}

func New(ctx context.Context, cfg *config.DatabaseConfig, logger *slog.Logger) (*DB, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.GetDSN())
	if err != nil {
		return nil, fmt.Errorf("failed to parse database config: %w", err)
	}

	poolConfig.HealthCheckPeriod = 1 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info("database connection established",
		slog.String("host", cfg.Host),
		slog.Int("port", cfg.Port),
		slog.String("database", cfg.DBName),
	)

	return &DB{
		Pool:   pool,
		logger: logger,
	}, nil
}

func (db *DB) Close() {
	if db.Pool != nil {
		db.logger.Info("closing database connection")
		db.Pool.Close()
	}
}

func (db *DB) Health(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if err := db.Pool.Ping(ctx); err != nil {
		return fmt.Errorf("database health check failed: %w", err)
	}

	return nil
}
