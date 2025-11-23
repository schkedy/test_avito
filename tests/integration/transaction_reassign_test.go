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

// TestReassignTransactions tests for reviewer reassignment transactions
func TestReassignTransactions(t *testing.T) {
	t.Run("ReassignReviewer_Success", func(t *testing.T) {
		pool, cleanup := setupTestDB(t)
		defer cleanup()

		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		teamRepo := repository.NewTeamRepository(pool, logger)
		prRepo := repository.NewPullRequestRepository(pool, logger)

		// Create team with members
		team := &domain.Team{
			Name: "test-team-reassign",
			Members: []domain.User{
				{ID: "author", Username: "author", IsActive: true},
				{ID: "reviewer1", Username: "reviewer1", IsActive: true},
				{ID: "reviewer2", Username: "reviewer2", IsActive: true},
			},
		}
		err := teamRepo.CreateWithMembers(context.Background(), team)
		require.NoError(t, err)

		// Create PR with reviewer1
		pr := &domain.PullRequest{
			ID:                "pr-reassign",
			Name:              "Test PR",
			AuthorID:          "author",
			Status:            domain.PRStatusOpen,
			AssignedReviewers: []string{"reviewer1"},
		}
		err = prRepo.Create(context.Background(), pr)
		require.NoError(t, err)

		// Reassign from reviewer1 to reviewer2
		err = prRepo.ReassignReviewer(context.Background(), "pr-reassign", "reviewer1", "reviewer2")
		require.NoError(t, err)

		// Verify reassignment
		updatedPR, err := prRepo.GetByID(context.Background(), "pr-reassign")
		require.NoError(t, err)
		assert.Len(t, updatedPR.AssignedReviewers, 1)
		assert.Equal(t, "reviewer2", updatedPR.AssignedReviewers[0])
	})

	t.Run("ReassignReviewer_Timeout", func(t *testing.T) {
		pool, cleanup := setupTestDB(t)
		defer cleanup()

		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		prRepo := repository.NewPullRequestRepository(pool, logger)

		// Create context with very short timeout
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()
		time.Sleep(10 * time.Millisecond) // Ensure timeout expires

		err := prRepo.ReassignReviewer(ctx, "pr-any", "old", "new")
		assert.Error(t, err, "Expected timeout error")
		assert.Contains(t, err.Error(), "context")
	})

	t.Run("ReassignReviewer_RollbackOnError", func(t *testing.T) {
		pool, cleanup := setupTestDB(t)
		defer cleanup()

		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		teamRepo := repository.NewTeamRepository(pool, logger)
		prRepo := repository.NewPullRequestRepository(pool, logger)

		// Create team with members
		team := &domain.Team{
			Name: "test-team-reassign-rollback",
			Members: []domain.User{
				{ID: "author2", Username: "author", IsActive: true},
				{ID: "reviewer3", Username: "reviewer3", IsActive: true},
			},
		}
		err := teamRepo.CreateWithMembers(context.Background(), team)
		require.NoError(t, err)

		// Create PR with reviewer
		pr := &domain.PullRequest{
			ID:                "pr-reassign-rollback",
			Name:              "Test PR Rollback",
			AuthorID:          "author2",
			Status:            domain.PRStatusOpen,
			AssignedReviewers: []string{"reviewer3"},
		}
		err = prRepo.Create(context.Background(), pr)
		require.NoError(t, err)

		// Try to reassign to invalid reviewer (empty ID) - should fail and rollback
		err = prRepo.ReassignReviewer(context.Background(), "pr-reassign-rollback", "reviewer3", "")
		assert.Error(t, err, "Expected error due to invalid new reviewer")

		// Verify original reviewer is still assigned (transaction rolled back)
		unchangedPR, err := prRepo.GetByID(context.Background(), "pr-reassign-rollback")
		require.NoError(t, err)
		assert.Len(t, unchangedPR.AssignedReviewers, 1)
		assert.Equal(t, "reviewer3", unchangedPR.AssignedReviewers[0], "Original reviewer should still be assigned")
	})

	t.Run("ReassignReviewer_ConcurrentReassignments", func(t *testing.T) {
		pool, cleanup := setupTestDB(t)
		defer cleanup()

		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		teamRepo := repository.NewTeamRepository(pool, logger)
		prRepo := repository.NewPullRequestRepository(pool, logger)

		// Create team with many reviewers
		members := []domain.User{
			{ID: "author3", Username: "author", IsActive: true},
		}
		for i := 1; i <= 10; i++ {
			members = append(members, domain.User{
				ID:       fmt.Sprintf("rev%d", i),
				Username: fmt.Sprintf("reviewer%d", i),
				IsActive: true,
			})
		}
		team := &domain.Team{
			Name:    "test-team-concurrent-reassign",
			Members: members,
		}
		err := teamRepo.CreateWithMembers(context.Background(), team)
		require.NoError(t, err)

		// Create multiple PRs with different reviewers
		const numPRs = 5
		for i := 0; i < numPRs; i++ {
			pr := &domain.PullRequest{
				ID:                fmt.Sprintf("pr-concurrent-reassign-%d", i),
				Name:              fmt.Sprintf("Test PR %d", i),
				AuthorID:          "author3",
				Status:            domain.PRStatusOpen,
				AssignedReviewers: []string{fmt.Sprintf("rev%d", i+1)},
			}
			err = prRepo.Create(context.Background(), pr)
			require.NoError(t, err)
		}

		// Concurrently reassign reviewers on different PRs
		var wg sync.WaitGroup
		errors := make([]error, numPRs)

		for i := 0; i < numPRs; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				prID := fmt.Sprintf("pr-concurrent-reassign-%d", idx)
				oldReviewer := fmt.Sprintf("rev%d", idx+1)
				newReviewer := fmt.Sprintf("rev%d", idx+6) // Different reviewer
				errors[idx] = prRepo.ReassignReviewer(context.Background(), prID, oldReviewer, newReviewer)
			}(i)
		}

		wg.Wait()

		// All reassignments should succeed (no race conditions)
		for i, err := range errors {
			assert.NoError(t, err, "Reassignment %d failed", i)
		}

		// Verify all reassignments were successful
		for i := 0; i < numPRs; i++ {
			pr, err := prRepo.GetByID(context.Background(), fmt.Sprintf("pr-concurrent-reassign-%d", i))
			require.NoError(t, err)
			expectedReviewer := fmt.Sprintf("rev%d", i+6)
			assert.Contains(t, pr.AssignedReviewers, expectedReviewer, "PR %d should have new reviewer", i)
		}
	})

	t.Run("ReassignReviewer_AtomicOperation", func(t *testing.T) {
		pool, cleanup := setupTestDB(t)
		defer cleanup()

		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		teamRepo := repository.NewTeamRepository(pool, logger)
		prRepo := repository.NewPullRequestRepository(pool, logger)

		// Create team with members
		team := &domain.Team{
			Name: "test-team-atomic",
			Members: []domain.User{
				{ID: "author4", Username: "author", IsActive: true},
				{ID: "reviewer4", Username: "reviewer4", IsActive: true},
				{ID: "reviewer5", Username: "reviewer5", IsActive: true},
			},
		}
		err := teamRepo.CreateWithMembers(context.Background(), team)
		require.NoError(t, err)

		// Create PR
		pr := &domain.PullRequest{
			ID:                "pr-atomic",
			Name:              "Test Atomic",
			AuthorID:          "author4",
			Status:            domain.PRStatusOpen,
			AssignedReviewers: []string{"reviewer4"},
		}
		err = prRepo.Create(context.Background(), pr)
		require.NoError(t, err)

		// Reassign reviewer
		err = prRepo.ReassignReviewer(context.Background(), "pr-atomic", "reviewer4", "reviewer5")
		require.NoError(t, err)

		// Verify atomicity - exactly one reviewer, the new one
		finalPR, err := prRepo.GetByID(context.Background(), "pr-atomic")
		require.NoError(t, err)
		assert.Len(t, finalPR.AssignedReviewers, 1, "Should have exactly 1 reviewer")
		assert.Equal(t, "reviewer5", finalPR.AssignedReviewers[0], "Should be the new reviewer")

		// Verify old reviewer is completely removed
		assert.NotContains(t, finalPR.AssignedReviewers, "reviewer4", "Old reviewer should be removed")
	})
}
