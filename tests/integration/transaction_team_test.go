package integration

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"test_avito/internal/domain"
	"test_avito/internal/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTeamTransactions tests for team creation and update transactions
func TestTeamTransactions(t *testing.T) {
	t.Run("CreateWithMembers_Success", func(t *testing.T) {
		pool, cleanup := setupTestDB(t)
		defer cleanup()

		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		repo := repository.NewTeamRepository(pool, logger)

		team := &domain.Team{
			Name: "test-team-1",
			Members: []domain.User{
				{ID: "user1", Username: "alice", IsActive: true},
				{ID: "user2", Username: "bob", IsActive: true},
				{ID: "user3", Username: "charlie", IsActive: true},
			},
		}

		err := repo.CreateWithMembers(context.Background(), team)
		require.NoError(t, err)

		// Verify team and members were created
		createdTeam, err := repo.GetByName(context.Background(), "test-team-1")
		require.NoError(t, err)
		assert.Equal(t, "test-team-1", createdTeam.Name)
		assert.Len(t, createdTeam.Members, 3)
	})

	t.Run("CreateWithMembers_Timeout", func(t *testing.T) {
		pool, cleanup := setupTestDB(t)
		defer cleanup()

		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		repo := repository.NewTeamRepository(pool, logger)

		// Create context with very short timeout
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()
		time.Sleep(10 * time.Millisecond) // Ensure timeout expires

		team := &domain.Team{
			Name: "test-team-timeout",
			Members: []domain.User{
				{ID: "user1", Username: "alice", IsActive: true},
			},
		}

		err := repo.CreateWithMembers(ctx, team)
		assert.Error(t, err, "Expected timeout error")
		assert.Contains(t, err.Error(), "context")
	})

	t.Run("CreateWithMembers_DuplicateTeam", func(t *testing.T) {
		pool, cleanup := setupTestDB(t)
		defer cleanup()

		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		repo := repository.NewTeamRepository(pool, logger)

		team := &domain.Team{
			Name: "duplicate-team",
			Members: []domain.User{
				{ID: "user1", Username: "alice", IsActive: true},
			},
		}

		// First creation should succeed
		err := repo.CreateWithMembers(context.Background(), team)
		require.NoError(t, err)

		// Second creation should fail due to unique constraint
		err = repo.CreateWithMembers(context.Background(), team)
		assert.Error(t, err, "Expected duplicate key error")
	})

	t.Run("CreateWithMembers_RollbackOnError", func(t *testing.T) {
		pool, cleanup := setupTestDB(t)
		defer cleanup()

		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		repo := repository.NewTeamRepository(pool, logger)

		// First create a team to establish a duplicate scenario
		firstTeam := &domain.Team{
			Name: "test-team-rollback",
			Members: []domain.User{
				{ID: "user1", Username: "alice", IsActive: true},
			},
		}
		err := repo.CreateWithMembers(context.Background(), firstTeam)
		require.NoError(t, err)

		// Try to create duplicate team - should fail on team creation
		// This will cause rollback, so members won't be inserted
		duplicateTeam := &domain.Team{
			Name: "test-team-rollback", // Same name - will violate unique constraint
			Members: []domain.User{
				{ID: "user2", Username: "bob", IsActive: true},
			},
		}

		err = repo.CreateWithMembers(context.Background(), duplicateTeam)
		assert.Error(t, err, "Expected error due to duplicate team")

		// Verify user2 was NOT created (transaction rolled back)
		createdTeam, err := repo.GetByName(context.Background(), "test-team-rollback")
		require.NoError(t, err)
		assert.Len(t, createdTeam.Members, 1)
		assert.Equal(t, "user1", createdTeam.Members[0].ID)
	})

	t.Run("UpdateMembers_Success", func(t *testing.T) {
		pool, cleanup := setupTestDB(t)
		defer cleanup()

		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		repo := repository.NewTeamRepository(pool, logger)

		// Create team first
		team := &domain.Team{
			Name: "test-team-update",
			Members: []domain.User{
				{ID: "user1", Username: "alice", IsActive: true},
			},
		}
		err := repo.CreateWithMembers(context.Background(), team)
		require.NoError(t, err)

		// Update members - UPSERT will add new members
		newMembers := []domain.User{
			{ID: "user2", Username: "bob", TeamName: "test-team-update", IsActive: true},
			{ID: "user3", Username: "charlie", TeamName: "test-team-update", IsActive: true},
		}
		err = repo.UpdateMembers(context.Background(), "test-team-update", newMembers)
		require.NoError(t, err)

		// Verify members were added (UPSERT adds, doesn't replace)
		updatedTeam, err := repo.GetByName(context.Background(), "test-team-update")
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(updatedTeam.Members), 2, "Should have at least 2 members")
	})

	t.Run("UpdateMembers_ConcurrentUpdates", func(t *testing.T) {
		pool, cleanup := setupTestDB(t)
		defer cleanup()

		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		repo := repository.NewTeamRepository(pool, logger)

		// Create team
		team := &domain.Team{
			Name: "test-team-concurrent",
			Members: []domain.User{
				{ID: "user1", Username: "alice", IsActive: true},
			},
		}
		err := repo.CreateWithMembers(context.Background(), team)
		require.NoError(t, err)

		// Run concurrent updates
		const numGoroutines = 10
		var wg sync.WaitGroup
		errors := make([]error, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				members := []domain.User{
					{ID: string(rune('a' + idx)), Username: string(rune('a' + idx)), IsActive: true},
				}
				errors[idx] = repo.UpdateMembers(context.Background(), "test-team-concurrent", members)
			}(i)
		}

		wg.Wait()

		// All updates should succeed (no deadlocks due to sorting)
		for i, err := range errors {
			assert.NoError(t, err, "Update %d failed", i)
		}
	})
}
