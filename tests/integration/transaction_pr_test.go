package integration

import (
	"context"
	"fmt"
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

// TestPRTransactions tests for pull request creation and merge transactions
func TestPRTransactions(t *testing.T) {
	t.Run("Create_Success", func(t *testing.T) {
		pool, cleanup := setupTestDB(t)
		defer cleanup()

		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		teamRepo := repository.NewTeamRepository(pool, logger)
		prRepo := repository.NewPullRequestRepository(pool, logger)

		// Create team with members
		team := &domain.Team{
			Name: "test-team-pr",
			Members: []domain.User{
				{ID: "author1", Username: "author", IsActive: true},
				{ID: "reviewer1", Username: "reviewer1", IsActive: true},
				{ID: "reviewer2", Username: "reviewer2", IsActive: true},
			},
		}
		err := teamRepo.CreateWithMembers(context.Background(), team)
		require.NoError(t, err)

		// Create PR with reviewers
		pr := &domain.PullRequest{
			ID:                "pr-1",
			Name:              "Test PR",
			AuthorID:          "author1",
			Status:            domain.PRStatusOpen,
			AssignedReviewers: []string{"reviewer1", "reviewer2"},
		}

		err = prRepo.Create(context.Background(), pr)
		require.NoError(t, err)

		// Verify PR was created
		createdPR, err := prRepo.GetByID(context.Background(), "pr-1")
		require.NoError(t, err)
		assert.Equal(t, "Test PR", createdPR.Name)
		assert.Len(t, createdPR.AssignedReviewers, 2)
	})

	t.Run("Create_Timeout", func(t *testing.T) {
		pool, cleanup := setupTestDB(t)
		defer cleanup()

		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		prRepo := repository.NewPullRequestRepository(pool, logger)

		// Create context with very short timeout
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()
		time.Sleep(10 * time.Millisecond) // Ensure timeout expires

		pr := &domain.PullRequest{
			ID:                "pr-timeout",
			Name:              "Test PR",
			AuthorID:          "author1",
			Status:            domain.PRStatusOpen,
			AssignedReviewers: []string{"reviewer1"},
		}

		err := prRepo.Create(ctx, pr)
		assert.Error(t, err, "Expected timeout error")
	})

	t.Run("Create_RollbackOnReviewerError", func(t *testing.T) {
		pool, cleanup := setupTestDB(t)
		defer cleanup()

		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		teamRepo := repository.NewTeamRepository(pool, logger)
		prRepo := repository.NewPullRequestRepository(pool, logger)

		// Create team with author
		team := &domain.Team{
			Name: "test-team-pr-rollback",
			Members: []domain.User{
				{ID: "author2", Username: "author", IsActive: true},
			},
		}
		err := teamRepo.CreateWithMembers(context.Background(), team)
		require.NoError(t, err)

		// Try to create PR with invalid reviewer (empty ID)
		pr := &domain.PullRequest{
			ID:                "pr-rollback",
			Name:              "Test PR",
			AuthorID:          "author2",
			Status:            domain.PRStatusOpen,
			AssignedReviewers: []string{""}, // Invalid empty reviewer ID
		}

		err = prRepo.Create(context.Background(), pr)
		assert.Error(t, err, "Expected error due to invalid reviewer")

		// Verify PR was NOT created (transaction rolled back)
		_, err = prRepo.GetByID(context.Background(), "pr-rollback")
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrPRNotFound)
	})

	t.Run("Create_DeadlockPrevention", func(t *testing.T) {
		pool, cleanup := setupTestDB(t)
		defer cleanup()

		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		teamRepo := repository.NewTeamRepository(pool, logger)
		prRepo := repository.NewPullRequestRepository(pool, logger)

		// Create team with reviewers
		team := &domain.Team{
			Name: "test-team-deadlock",
			Members: []domain.User{
				{ID: "author3", Username: "author", IsActive: true},
				{ID: "rev1", Username: "reviewer1", IsActive: true},
				{ID: "rev2", Username: "reviewer2", IsActive: true},
				{ID: "rev3", Username: "reviewer3", IsActive: true},
			},
		}
		err := teamRepo.CreateWithMembers(context.Background(), team)
		require.NoError(t, err)

		// Create PRs concurrently with different reviewer orders
		const numGoroutines = 10
		var wg sync.WaitGroup
		errors := make([]error, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				pr := &domain.PullRequest{
					ID:       fmt.Sprintf("pr-deadlock-%d", idx),
					Name:     fmt.Sprintf("Test PR %d", idx),
					AuthorID: "author3",
					Status:   domain.PRStatusOpen,
					// Different orders - but will be sorted internally
					AssignedReviewers: []string{"rev3", "rev1", "rev2"},
				}
				if idx%2 == 0 {
					pr.AssignedReviewers = []string{"rev2", "rev3", "rev1"}
				}
				errors[idx] = prRepo.Create(context.Background(), pr)
			}(i)
		}

		wg.Wait()

		// All creates should succeed (no deadlocks due to sorting)
		for i, err := range errors {
			assert.NoError(t, err, "Create %d failed", i)
		}
	})

	t.Run("Merge_Idempotent", func(t *testing.T) {
		pool, cleanup := setupTestDB(t)
		defer cleanup()

		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		teamRepo := repository.NewTeamRepository(pool, logger)
		prRepo := repository.NewPullRequestRepository(pool, logger)

		// Create team and author
		team := &domain.Team{
			Name: "test-team-merge",
			Members: []domain.User{
				{ID: "author4", Username: "author", IsActive: true},
			},
		}
		err := teamRepo.CreateWithMembers(context.Background(), team)
		require.NoError(t, err)

		// Create PR
		pr := &domain.PullRequest{
			ID:                "pr-merge",
			Name:              "Test PR Merge",
			AuthorID:          "author4",
			Status:            domain.PRStatusOpen,
			AssignedReviewers: []string{},
		}
		err = prRepo.Create(context.Background(), pr)
		require.NoError(t, err)

		// Merge PR first time
		mergedPR1, err := prRepo.Merge(context.Background(), "pr-merge")
		require.NoError(t, err)
		assert.Equal(t, domain.PRStatusMerged, mergedPR1.Status)
		assert.NotNil(t, mergedPR1.MergedAt)

		// Merge PR second time (should be idempotent)
		mergedPR2, err := prRepo.Merge(context.Background(), "pr-merge")
		require.NoError(t, err)
		assert.Equal(t, domain.PRStatusMerged, mergedPR2.Status)
		assert.NotNil(t, mergedPR2.MergedAt)

		// Both should have the same merged timestamp (idempotent)
		assert.Equal(t, mergedPR1.MergedAt.Unix(), mergedPR2.MergedAt.Unix())
	})

	t.Run("Merge_NonExistent", func(t *testing.T) {
		pool, cleanup := setupTestDB(t)
		defer cleanup()

		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		prRepo := repository.NewPullRequestRepository(pool, logger)

		// Try to merge non-existent PR
		_, err := prRepo.Merge(context.Background(), "non-existent-pr")
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrPRNotFound)
	})
}
