// Имплементация репозитория для работы с пользователями в базе данных postgresql
package repository

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"test_avito/internal/database/db"
	"test_avito/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepositoryImpl struct {
	queries *db.Queries
	pool    *pgxpool.Pool
	logger  *slog.Logger
}

func NewUserRepository(pool *pgxpool.Pool, logger *slog.Logger) *UserRepositoryImpl {
	return &UserRepositoryImpl{
		queries: db.New(pool),
		pool:    pool,
		logger:  logger,
	}
}

// Create creates a new user
func (r *UserRepositoryImpl) Create(ctx context.Context, user *domain.User) error {
	err := r.queries.CreateUser(ctx, db.CreateUserParams{
		ID:       user.ID,
		Username: user.Username,
		TeamName: user.TeamName,
		IsActive: user.IsActive,
	})
	if err != nil {
		r.logger.Error("failed to create user",
			slog.String("user_id", user.ID),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to create user: %w", err)
	}

	r.logger.Info("user created", slog.String("user_id", user.ID))
	return nil
}

// Update updates an existing user
func (r *UserRepositoryImpl) Update(ctx context.Context, user *domain.User) error {
	err := r.queries.UpdateUser(ctx, db.UpdateUserParams{
		ID:       user.ID,
		Username: user.Username,
		TeamName: user.TeamName,
		IsActive: user.IsActive,
	})
	if err != nil {
		r.logger.Error("failed to update user",
			slog.String("user_id", user.ID),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to update user: %w", err)
	}

	r.logger.Info("user updated", slog.String("user_id", user.ID))
	return nil
}

// Upsert creates or updates a user
func (r *UserRepositoryImpl) Upsert(ctx context.Context, user *domain.User) error {
	err := r.queries.UpsertUser(ctx, db.UpsertUserParams{
		ID:       user.ID,
		Username: user.Username,
		TeamName: user.TeamName,
		IsActive: user.IsActive,
	})
	if err != nil {
		r.logger.Error("failed to upsert user",
			slog.String("user_id", user.ID),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to upsert user: %w", err)
	}

	r.logger.Info("user upserted", slog.String("user_id", user.ID))
	return nil
}

// GetByID retrieves a user by ID
func (r *UserRepositoryImpl) GetByID(ctx context.Context, id string) (*domain.User, error) {
	dbUser, err := r.queries.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		r.logger.Error("failed to get user",
			slog.String("user_id", id),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &domain.User{
		ID:       dbUser.ID,
		Username: dbUser.Username,
		TeamName: dbUser.TeamName,
		IsActive: dbUser.IsActive,
	}, nil
}

// GetByTeam retrieves all users in a team
func (r *UserRepositoryImpl) GetByTeam(ctx context.Context, teamName string) ([]domain.User, error) {
	dbUsers, err := r.queries.GetUsersByTeam(ctx, teamName)
	if err != nil {
		r.logger.Error("failed to get users by team",
			slog.String("team_name", teamName),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("failed to get users by team: %w", err)
	}

	users := make([]domain.User, len(dbUsers))
	for i, u := range dbUsers {
		users[i] = domain.User{
			ID:       u.ID,
			Username: u.Username,
			TeamName: u.TeamName,
			IsActive: u.IsActive,
		}
	}

	return users, nil
}

// SetIsActive updates the user's active status
func (r *UserRepositoryImpl) SetIsActive(ctx context.Context, userID string, isActive bool) error {
	err := r.queries.SetUserIsActive(ctx, db.SetUserIsActiveParams{
		ID:       userID,
		IsActive: isActive,
	})
	if err != nil {
		r.logger.Error("failed to set user active status",
			slog.String("user_id", userID),
			slog.Bool("is_active", isActive),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to set user active status: %w", err)
	}

	r.logger.Info("user active status updated",
		slog.String("user_id", userID),
		slog.Bool("is_active", isActive),
	)
	return nil
}

// GetActiveByTeam retrieves all active users in a team excluding specific user
func (r *UserRepositoryImpl) GetActiveByTeam(ctx context.Context, teamName string, excludeUserID string) ([]domain.User, error) {
	dbUsers, err := r.queries.GetActiveUsersByTeam(ctx, db.GetActiveUsersByTeamParams{
		TeamName: teamName,
		ID:       excludeUserID,
	})
	if err != nil {
		r.logger.Error("failed to get active users by team",
			slog.String("team_name", teamName),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("failed to get active users by team: %w", err)
	}

	users := make([]domain.User, len(dbUsers))
	for i, u := range dbUsers {
		users[i] = domain.User{
			ID:       u.ID,
			Username: u.Username,
			TeamName: u.TeamName,
			IsActive: u.IsActive,
		}
	}

	return users, nil
}

// Exists checks if a user exists
func (r *UserRepositoryImpl) Exists(ctx context.Context, id string) (bool, error) {
	exists, err := r.queries.UserExists(ctx, id)
	if err != nil {
		r.logger.Error("failed to check user existence",
			slog.String("user_id", id),
			slog.String("error", err.Error()),
		)
		return false, fmt.Errorf("failed to check user existence: %w", err)
	}

	return exists, nil
}

// Count returns the total number of users
func (r *UserRepositoryImpl) Count(ctx context.Context) (int, error) {
	count, err := r.queries.CountUsers(ctx)
	if err != nil {
		r.logger.Error("failed to count users", slog.String("error", err.Error()))
		return 0, fmt.Errorf("failed to count users: %w", err)
	}

	return int(count), nil
}

// CountActive returns the number of active users
func (r *UserRepositoryImpl) CountActive(ctx context.Context) (int, error) {
	count, err := r.queries.CountActiveUsers(ctx)
	if err != nil {
		r.logger.Error("failed to count active users", slog.String("error", err.Error()))
		return 0, fmt.Errorf("failed to count active users: %w", err)
	}

	return int(count), nil
}

// DeactivateTeamUsers deactivates all users in a team (atomic operation)
func (r *UserRepositoryImpl) DeactivateTeamUsers(ctx context.Context, teamName string) (int, error) {
	rowsAffected, err := r.queries.DeactivateTeamUsers(ctx, teamName)
	if err != nil {
		r.logger.Error("failed to deactivate team users",
			slog.String("team_name", teamName),
			slog.String("error", err.Error()),
		)
		return 0, fmt.Errorf("failed to deactivate team users: %w", err)
	}

	r.logger.Info("team users deactivated",
		slog.String("team_name", teamName),
		slog.Int64("count", rowsAffected),
	)
	return int(rowsAffected), nil
}
