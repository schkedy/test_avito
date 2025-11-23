// Имплементация репозитория для получения статистики из базы данных postgresql
package repository

import (
	"context"
	"fmt"
	"log/slog"

	"test_avito/internal/database/db"

	"github.com/jackc/pgx/v5/pgxpool"
)

type StatsRepositoryImpl struct {
	queries *db.Queries
	logger  *slog.Logger
}

func NewStatsRepository(pool *pgxpool.Pool, logger *slog.Logger) *StatsRepositoryImpl {
	return &StatsRepositoryImpl{
		queries: db.New(pool),
		logger:  logger,
	}
}

func (r *StatsRepositoryImpl) GetStats(ctx context.Context) (*Stats, error) {
	dbStats, err := r.queries.GetStats(ctx)
	if err != nil {
		r.logger.Error("failed to get stats", slog.String("error", err.Error()))
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	return &Stats{
		TotalPRs:    int(dbStats.TotalPrs),
		OpenPRs:     int(dbStats.OpenPrs),
		MergedPRs:   int(dbStats.MergedPrs),
		TotalTeams:  int(dbStats.TotalTeams),
		TotalUsers:  int(dbStats.TotalUsers),
		ActiveUsers: int(dbStats.ActiveUsers),
	}, nil
}
