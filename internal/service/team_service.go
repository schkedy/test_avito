package service

import (
	"context"
	"fmt"
	"log/slog"

	"test_avito/internal/domain"
	"test_avito/internal/repository"
)

type TeamService struct {
	teamRepo repository.TeamRepository
	userRepo repository.UserRepository
	logger   *slog.Logger
}

func NewTeamService(
	teamRepo repository.TeamRepository,
	userRepo repository.UserRepository,
	logger *slog.Logger,
) *TeamService {
	return &TeamService{
		teamRepo: teamRepo,
		userRepo: userRepo,
		logger:   logger,
	}
}

// AddTeam creates or updates a team with members
// If team exists, updates members (upsert)
func (s *TeamService) AddTeam(ctx context.Context, team *domain.Team) error {
	if err := team.Validate(); err != nil {
		return err
	}

	s.logger.Info("adding team",
		slog.String("team_name", team.Name),
		slog.Int("members_count", len(team.Members)),
	)

	for i := range team.Members {
		team.Members[i].TeamName = team.Name
		if err := team.Members[i].Validate(); err != nil {
			s.logger.Warn("invalid member data",
				slog.String("user_id", team.Members[i].ID),
				slog.String("error", err.Error()),
			)
			return fmt.Errorf("invalid member %s: %w", team.Members[i].ID, err)
		}
	}

	exists, err := s.teamRepo.Exists(ctx, team.Name)
	if err != nil {
		return fmt.Errorf("failed to check team existence: %w", err)
	}

	if !exists {
		if err := s.teamRepo.CreateWithMembers(ctx, team); err != nil {
			return fmt.Errorf("failed to create team with members: %w", err)
		}
		s.logger.Info("team created with members",
			slog.String("team_name", team.Name),
			slog.Int("members_count", len(team.Members)),
		)
	} else {
		if err := s.teamRepo.UpdateMembers(ctx, team.Name, team.Members); err != nil {
			return fmt.Errorf("failed to update team members: %w", err)
		}
		s.logger.Info("team members updated",
			slog.String("team_name", team.Name),
			slog.Int("members_count", len(team.Members)),
		)
	}

	return nil
}

// GetTeam retrieves a team by name
func (s *TeamService) GetTeam(ctx context.Context, name string) (*domain.Team, error) {
	if name == "" {
		return nil, domain.ErrInvalidInput
	}

	team, err := s.teamRepo.GetByName(ctx, name)
	if err != nil {
		return nil, err
	}

	s.logger.Info("team retrieved",
		slog.String("team_name", name),
		slog.Int("members_count", len(team.Members)),
	)

	return team, nil
}

// DeactivateTeam deactivates all users in a team
func (s *TeamService) DeactivateTeam(ctx context.Context, teamName string) (*domain.Team, int, error) {
	if teamName == "" {
		return nil, 0, domain.ErrInvalidInput
	}

	exists, err := s.teamRepo.Exists(ctx, teamName)
	if err != nil {
		return nil, 0, err
	}
	if !exists {
		return nil, 0, domain.ErrTeamNotFound
	}

	deactivatedCount, err := s.userRepo.DeactivateTeamUsers(ctx, teamName)
	if err != nil {
		return nil, 0, err
	}

	team, err := s.teamRepo.GetByName(ctx, teamName)
	if err != nil {
		return nil, 0, err
	}

	s.logger.Info("team deactivated",
		slog.String("team_name", teamName),
		slog.Int("deactivated_count", deactivatedCount),
	)

	return team, deactivatedCount, nil
}
