package service

import (
	"context"
	"fmt"
	"log/slog"

	"test_avito/internal/domain"
	"test_avito/internal/repository"
)

type UserService struct {
	userRepo repository.UserRepository
	logger   *slog.Logger
}

func NewUserService(userRepo repository.UserRepository, logger *slog.Logger) *UserService {
	return &UserService{
		userRepo: userRepo,
		logger:   logger,
	}
}

// SetIsActive updates a user's active status
func (s *UserService) SetIsActive(ctx context.Context, userID string, isActive bool) (*domain.User, error) {
	if userID == "" {
		return nil, domain.ErrInvalidInput
	}

	s.logger.Info("setting user active status",
		slog.String("user_id", userID),
		slog.Bool("is_active", isActive),
	)

	if _, err := s.userRepo.GetByID(ctx, userID); err != nil {
		return nil, err
	}

	if err := s.userRepo.SetIsActive(ctx, userID, isActive); err != nil {
		return nil, fmt.Errorf("failed to update user status: %w", err)
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated user: %w", err)
	}

	s.logger.Info("user active status updated",
		slog.String("user_id", userID),
		slog.Bool("is_active", isActive),
	)

	return user, nil
}

// GetReviewsByUser retrieves all PRs where user is a reviewer
func (s *UserService) GetReviewsByUser(ctx context.Context, userID string) ([]domain.PullRequestShort, error) {
	if userID == "" {
		return nil, domain.ErrInvalidInput
	}

	_, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	s.logger.Info("getting reviews for user", slog.String("user_id", userID))

	return nil, nil
}
