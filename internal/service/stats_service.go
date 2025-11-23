package service

import (
	"context"
	"log/slog"

	"test_avito/internal/repository"
)

type StatsService struct {
	statsRepo repository.StatsRepository
	logger    *slog.Logger
}

func NewStatsService(statsRepo repository.StatsRepository, logger *slog.Logger) *StatsService {
	return &StatsService{
		statsRepo: statsRepo,
		logger:    logger,
	}
}

func (s *StatsService) GetStats(ctx context.Context) (*repository.Stats, error) {
	s.logger.Info("retrieving stats")

	stats, err := s.statsRepo.GetStats(ctx)
	if err != nil {
		return nil, err
	}

	s.logger.Info("stats retrieved",
		slog.Int("total_prs", stats.TotalPRs),
		slog.Int("total_teams", stats.TotalTeams),
		slog.Int("total_users", stats.TotalUsers),
	)

	return stats, nil
}
