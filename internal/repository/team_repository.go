// Имплементация репозитория для работы с pull request'ами в базе данных postgresql
package repository

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"test_avito/internal/database/db"
	"test_avito/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TeamRepositoryImpl struct {
	queries *db.Queries
	pool    *pgxpool.Pool
	logger  *slog.Logger
}

func NewTeamRepository(pool *pgxpool.Pool, logger *slog.Logger) *TeamRepositoryImpl {
	return &TeamRepositoryImpl{
		queries: db.New(pool),
		pool:    pool,
		logger:  logger,
	}
}

func (r *TeamRepositoryImpl) Create(ctx context.Context, team *domain.Team) error {
	err := r.queries.CreateTeam(ctx, team.Name)
	if err != nil {
		r.logger.Error("failed to create team",
			slog.String("team_name", team.Name),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to create team: %w", err)
	}

	r.logger.Info("team created", slog.String("team_name", team.Name))
	return nil
}

// CreateWithMembers creates a team with members in a transaction
func (r *TeamRepositoryImpl) CreateWithMembers(ctx context.Context, team *domain.Team) error {
	txCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	tx, err := r.pool.Begin(txCtx)
	if err != nil {
		r.logger.Error("failed to begin transaction",
			slog.String("team_name", team.Name),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(context.Background())
			r.logger.Error("panic in CreateWithMembers transaction",
				slog.String("team_name", team.Name),
				slog.Any("panic", p),
			)
			panic(p)
		}
		_ = tx.Rollback(context.Background())
	}()

	qtx := r.queries.WithTx(tx)

	err = qtx.CreateTeam(txCtx, team.Name)
	if err != nil {
		r.logger.Error("failed to create team in transaction",
			slog.String("team_name", team.Name),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to create team: %w", err)
	}

	members := make([]domain.User, len(team.Members))
	copy(members, team.Members)
	sort.Slice(members, func(i, j int) bool {
		return members[i].ID < members[j].ID
	})

	for _, member := range members {
		err = qtx.UpsertUser(txCtx, db.UpsertUserParams{
			ID:       member.ID,
			Username: member.Username,
			TeamName: team.Name,
			IsActive: member.IsActive,
		})
		if err != nil {
			r.logger.Error("failed to upsert member in transaction",
				slog.String("team_name", team.Name),
				slog.String("user_id", member.ID),
				slog.String("error", err.Error()),
			)
			return fmt.Errorf("failed to upsert member %s: %w", member.ID, err)
		}
	}

	if err := tx.Commit(txCtx); err != nil {
		r.logger.Error("failed to commit transaction",
			slog.String("team_name", team.Name),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	r.logger.Info("team created with members in transaction",
		slog.String("team_name", team.Name),
		slog.Int("members_count", len(team.Members)),
	)
	return nil
}

// UpdateMembers updates team members in a transaction
func (r *TeamRepositoryImpl) UpdateMembers(ctx context.Context, teamName string, members []domain.User) error {
	txCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	tx, err := r.pool.Begin(txCtx)
	if err != nil {
		r.logger.Error("failed to begin transaction",
			slog.String("team_name", teamName),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(context.Background())
			r.logger.Error("panic in UpdateMembers transaction",
				slog.String("team_name", teamName),
				slog.Any("panic", p),
			)
			panic(p)
		}
		_ = tx.Rollback(context.Background())
	}()

	qtx := r.queries.WithTx(tx)

	sortedMembers := make([]domain.User, len(members))
	copy(sortedMembers, members)
	sort.Slice(sortedMembers, func(i, j int) bool {
		return sortedMembers[i].ID < sortedMembers[j].ID
	})

	for _, member := range sortedMembers {
		err = qtx.UpsertUser(txCtx, db.UpsertUserParams{
			ID:       member.ID,
			Username: member.Username,
			TeamName: teamName,
			IsActive: member.IsActive,
		})
		if err != nil {
			r.logger.Error("failed to upsert member in transaction",
				slog.String("team_name", teamName),
				slog.String("user_id", member.ID),
				slog.String("error", err.Error()),
			)
			return fmt.Errorf("failed to upsert member %s: %w", member.ID, err)
		}
	}

	if err := tx.Commit(txCtx); err != nil {
		r.logger.Error("failed to commit transaction",
			slog.String("team_name", teamName),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	r.logger.Info("team members updated in transaction",
		slog.String("team_name", teamName),
		slog.Int("members_count", len(members)),
	)
	return nil
}

// GetByName retrieves a team by name with its members
func (r *TeamRepositoryImpl) GetByName(ctx context.Context, name string) (*domain.Team, error) {
	teamName, err := r.queries.GetTeamByName(ctx, name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrTeamNotFound
		}
		r.logger.Error("failed to get team",
			slog.String("team_name", name),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("failed to get team: %w", err)
	}

	dbUsers, err := r.queries.GetUsersByTeam(ctx, name)
	if err != nil {
		r.logger.Error("failed to get team members",
			slog.String("team_name", name),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("failed to get team members: %w", err)
	}

	members := make([]domain.User, len(dbUsers))
	for i, u := range dbUsers {
		members[i] = domain.User{
			ID:       u.ID,
			Username: u.Username,
			TeamName: u.TeamName,
			IsActive: u.IsActive,
		}
	}

	team := &domain.Team{
		Name:    teamName,
		Members: members,
	}

	return team, nil
}

// Exists checks if a team exists
func (r *TeamRepositoryImpl) Exists(ctx context.Context, name string) (bool, error) {
	exists, err := r.queries.TeamExists(ctx, name)
	if err != nil {
		r.logger.Error("failed to check team existence",
			slog.String("team_name", name),
			slog.String("error", err.Error()),
		)
		return false, fmt.Errorf("failed to check team existence: %w", err)
	}

	return exists, nil
}

// Count returns the total number of teams
func (r *TeamRepositoryImpl) Count(ctx context.Context) (int, error) {
	count, err := r.queries.CountTeams(ctx)
	if err != nil {
		r.logger.Error("failed to count teams", slog.String("error", err.Error()))
		return 0, fmt.Errorf("failed to count teams: %w", err)
	}

	return int(count), nil
}
