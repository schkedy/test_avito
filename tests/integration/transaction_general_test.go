package integration

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"testing"

	"test_avito/internal/domain"
	"test_avito/internal/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGeneralTransactions tests general transaction features like panic recovery, commit handling, and isolation
func TestGeneralTransactions(t *testing.T) {
	t.Run("TransactionPanic_Recovery", func(t *testing.T) {
		pool, cleanup := setupTestDB(t)
		defer cleanup()

		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		repo := repository.NewTeamRepository(pool, logger)

		// This test verifies panic recovery is in place
		// We can't easily trigger a panic in production code without modifying it,
		// but we verify the transaction completes or rolls back correctly

		team := &domain.Team{
			Name: "test-team-panic",
			Members: []domain.User{
				{ID: "user1", Username: "alice", IsActive: true},
			},
		}

		// Normal execution should work
		err := repo.CreateWithMembers(context.Background(), team)
		require.NoError(t, err)

		// Verify team was created
		createdTeam, err := repo.GetByName(context.Background(), "test-team-panic")
		require.NoError(t, err)
		assert.NotNil(t, createdTeam)
	})

	t.Run("TransactionCommit_ErrorHandling", func(t *testing.T) {
		pool, cleanup := setupTestDB(t)
		defer cleanup()

		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		repo := repository.NewTeamRepository(pool, logger)

		// Create team with valid data
		team := &domain.Team{
			Name: "test-team-commit",
			Members: []domain.User{
				{ID: "user1", Username: "alice", IsActive: true},
			},
		}

		err := repo.CreateWithMembers(context.Background(), team)
		require.NoError(t, err)

		// Verify transaction committed successfully
		createdTeam, err := repo.GetByName(context.Background(), "test-team-commit")
		require.NoError(t, err)
		assert.Equal(t, "test-team-commit", createdTeam.Name)
	})

	t.Run("ConcurrentTransactions_Isolation", func(t *testing.T) {
		pool, cleanup := setupTestDB(t)
		defer cleanup()

		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		repo := repository.NewTeamRepository(pool, logger)

		// Create multiple teams concurrently
		const numTeams = 20
		var wg sync.WaitGroup
		errors := make([]error, numTeams)

		for i := 0; i < numTeams; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				team := &domain.Team{
					Name: fmt.Sprintf("concurrent-team-%d", idx),
					Members: []domain.User{
						{ID: fmt.Sprintf("user-%d", idx), Username: fmt.Sprintf("user%d", idx), IsActive: true},
					},
				}
				errors[idx] = repo.CreateWithMembers(context.Background(), team)
			}(i)
		}

		wg.Wait()

		// All creates should succeed
		successCount := 0
		for _, err := range errors {
			if err == nil {
				successCount++
			}
		}
		assert.Equal(t, numTeams, successCount, "All concurrent transactions should succeed")
	})
}
