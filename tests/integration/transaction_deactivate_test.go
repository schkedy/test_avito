package integration

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"test_avito/internal/domain"
	"test_avito/internal/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDeactivateTransactions tests for user deactivation transactions
func TestDeactivateTransactions(t *testing.T) {
	t.Run("DeactivateTeamUsers_Success", func(t *testing.T) {
		pool, cleanup := setupTestDB(t)
		defer cleanup()

		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		teamRepo := repository.NewTeamRepository(pool, logger)
		userRepo := repository.NewUserRepository(pool, logger)

		// Create team with active members
		team := &domain.Team{
			Name: "test-team-deactivate",
			Members: []domain.User{
				{ID: "user-d1", Username: "user1", IsActive: true},
				{ID: "user-d2", Username: "user2", IsActive: true},
				{ID: "user-d3", Username: "user3", IsActive: true},
			},
		}
		err := teamRepo.CreateWithMembers(context.Background(), team)
		require.NoError(t, err)

		// Deactivate all team users
		count, err := userRepo.DeactivateTeamUsers(context.Background(), "test-team-deactivate")
		require.NoError(t, err)
		assert.Equal(t, 3, count)

		// Verify all users are deactivated
		updatedTeam, err := teamRepo.GetByName(context.Background(), "test-team-deactivate")
		require.NoError(t, err)
		for _, member := range updatedTeam.Members {
			assert.False(t, member.IsActive, "User %s should be deactivated", member.ID)
		}
	})

	t.Run("DeactivateTeamUsers_Timeout", func(t *testing.T) {
		pool, cleanup := setupTestDB(t)
		defer cleanup()

		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		userRepo := repository.NewUserRepository(pool, logger)

		// Create context with very short timeout
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()
		time.Sleep(10 * time.Millisecond) // Ensure timeout expires

		_, err := userRepo.DeactivateTeamUsers(ctx, "any-team")
		assert.Error(t, err, "Expected timeout error")
		assert.Contains(t, err.Error(), "context")
	})

	t.Run("DeactivateTeamUsers_EmptyTeam", func(t *testing.T) {
		pool, cleanup := setupTestDB(t)
		defer cleanup()

		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		teamRepo := repository.NewTeamRepository(pool, logger)
		userRepo := repository.NewUserRepository(pool, logger)

		// Create team without members
		team := &domain.Team{
			Name:    "test-team-empty",
			Members: []domain.User{},
		}
		err := teamRepo.Create(context.Background(), team)
		require.NoError(t, err)

		// Deactivate users in empty team
		count, err := userRepo.DeactivateTeamUsers(context.Background(), "test-team-empty")
		require.NoError(t, err)
		assert.Equal(t, 0, count, "Should deactivate 0 users")
	})

	t.Run("DeactivateTeamUsers_NonExistentTeam", func(t *testing.T) {
		pool, cleanup := setupTestDB(t)
		defer cleanup()

		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		userRepo := repository.NewUserRepository(pool, logger)

		// Deactivate users in non-existent team
		count, err := userRepo.DeactivateTeamUsers(context.Background(), "non-existent-team")
		require.NoError(t, err)
		assert.Equal(t, 0, count, "Should deactivate 0 users")
	})
}
