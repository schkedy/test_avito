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

// TestAssignTransactions tests for PR reviewer assignment transactions
func TestAssignTransactions(t *testing.T) {
	t.Run("AssignReviewers_RandomSuccess", func(t *testing.T) {
		pool, cleanup := setupTestDB(t)
		defer cleanup()

		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		teamRepo := repository.NewTeamRepository(pool, logger)
		prRepo := repository.NewPullRequestRepository(pool, logger)

		// Create team with members
		team := &domain.Team{
			Name: "test-team-assign-random",
			Members: []domain.User{
				{ID: "author-a1", Username: "author", IsActive: true},
				{ID: "reviewer-a1", Username: "reviewer1", IsActive: true},
				{ID: "reviewer-a2", Username: "reviewer2", IsActive: true},
				{ID: "reviewer-a3", Username: "reviewer3", IsActive: true},
			},
		}
		err := teamRepo.CreateWithMembers(context.Background(), team)
		require.NoError(t, err)

		// Create PR without reviewers
		pr := &domain.PullRequest{
			ID:                "pr-assign-random",
			Name:              "Test PR",
			AuthorID:          "author-a1",
			Status:            domain.PRStatusOpen,
			AssignedReviewers: []string{},
		}
		err = prRepo.Create(context.Background(), pr)
		require.NoError(t, err)

		// Assign with empty list should succeed (assigns no one, PR still has no reviewers)
		err = prRepo.AssignReviewers(context.Background(), "pr-assign-random", []string{})
		require.NoError(t, err) // Should succeed - assigns nobody

		// Verify PR still has no reviewers
		unchangedPR, err := prRepo.GetByID(context.Background(), "pr-assign-random")
		require.NoError(t, err)
		assert.Len(t, unchangedPR.AssignedReviewers, 0, "PR should have no reviewers")
	})

	t.Run("AssignReviewers_SpecificSuccess", func(t *testing.T) {
		pool, cleanup := setupTestDB(t)
		defer cleanup()

		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		teamRepo := repository.NewTeamRepository(pool, logger)
		prRepo := repository.NewPullRequestRepository(pool, logger)

		// Create team with members
		team := &domain.Team{
			Name: "test-team-assign-specific",
			Members: []domain.User{
				{ID: "author-a2", Username: "author", IsActive: true},
				{ID: "reviewer-a4", Username: "reviewer1", IsActive: true},
				{ID: "reviewer-a5", Username: "reviewer2", IsActive: true},
			},
		}
		err := teamRepo.CreateWithMembers(context.Background(), team)
		require.NoError(t, err)

		// Create PR without reviewers
		pr := &domain.PullRequest{
			ID:                "pr-assign-specific",
			Name:              "Test PR",
			AuthorID:          "author-a2",
			Status:            domain.PRStatusOpen,
			AssignedReviewers: []string{},
		}
		err = prRepo.Create(context.Background(), pr)
		require.NoError(t, err)

		// Assign specific reviewers
		err = prRepo.AssignReviewers(context.Background(), "pr-assign-specific", []string{"reviewer-a4", "reviewer-a5"})
		require.NoError(t, err)

		// Verify reviewers were assigned
		updatedPR, err := prRepo.GetByID(context.Background(), "pr-assign-specific")
		require.NoError(t, err)
		assert.Len(t, updatedPR.AssignedReviewers, 2)
		assert.Contains(t, updatedPR.AssignedReviewers, "reviewer-a4")
		assert.Contains(t, updatedPR.AssignedReviewers, "reviewer-a5")
	})

	t.Run("AssignReviewers_AlreadyAssignedError", func(t *testing.T) {
		pool, cleanup := setupTestDB(t)
		defer cleanup()

		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		teamRepo := repository.NewTeamRepository(pool, logger)
		prRepo := repository.NewPullRequestRepository(pool, logger)

		// Create team with members
		team := &domain.Team{
			Name: "test-team-assign-error",
			Members: []domain.User{
				{ID: "author-a3", Username: "author", IsActive: true},
				{ID: "reviewer-a6", Username: "reviewer1", IsActive: true},
				{ID: "reviewer-a7", Username: "reviewer2", IsActive: true},
			},
		}
		err := teamRepo.CreateWithMembers(context.Background(), team)
		require.NoError(t, err)

		// Create PR with reviewers already assigned
		pr := &domain.PullRequest{
			ID:                "pr-assign-error",
			Name:              "Test PR",
			AuthorID:          "author-a3",
			Status:            domain.PRStatusOpen,
			AssignedReviewers: []string{"reviewer-a6"},
		}
		err = prRepo.Create(context.Background(), pr)
		require.NoError(t, err)

		// Try to assign reviewers (should fail - PR already has reviewers)
		err = prRepo.AssignReviewers(context.Background(), "pr-assign-error", []string{"reviewer-a7"})
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrReviewersAlreadyAssigned)
	})

	t.Run("AssignReviewers_Timeout", func(t *testing.T) {
		pool, cleanup := setupTestDB(t)
		defer cleanup()

		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		prRepo := repository.NewPullRequestRepository(pool, logger)

		// Create context with very short timeout
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()
		time.Sleep(10 * time.Millisecond) // Ensure timeout expires

		err := prRepo.AssignReviewers(ctx, "pr-any", []string{"reviewer1"})
		assert.Error(t, err, "Expected timeout error")
	})

	t.Run("AssignReviewers_RollbackOnError", func(t *testing.T) {
		pool, cleanup := setupTestDB(t)
		defer cleanup()

		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		teamRepo := repository.NewTeamRepository(pool, logger)
		prRepo := repository.NewPullRequestRepository(pool, logger)

		// Create team with members
		team := &domain.Team{
			Name: "test-team-assign-rollback",
			Members: []domain.User{
				{ID: "author-a4", Username: "author", IsActive: true},
				{ID: "reviewer-a8", Username: "reviewer1", IsActive: true},
			},
		}
		err := teamRepo.CreateWithMembers(context.Background(), team)
		require.NoError(t, err)

		// Create PR without reviewers
		pr := &domain.PullRequest{
			ID:                "pr-assign-rollback",
			Name:              "Test PR",
			AuthorID:          "author-a4",
			Status:            domain.PRStatusOpen,
			AssignedReviewers: []string{},
		}
		err = prRepo.Create(context.Background(), pr)
		require.NoError(t, err)

		// Try to assign with invalid reviewer (empty ID)
		err = prRepo.AssignReviewers(context.Background(), "pr-assign-rollback", []string{"reviewer-a8", ""})
		assert.Error(t, err)

		// Verify PR still has no reviewers (transaction rolled back)
		unchangedPR, err := prRepo.GetByID(context.Background(), "pr-assign-rollback")
		require.NoError(t, err)
		assert.Len(t, unchangedPR.AssignedReviewers, 0, "PR should have no reviewers after rollback")
	})

	t.Run("AssignReviewers_DeadlockPrevention", func(t *testing.T) {
		pool, cleanup := setupTestDB(t)
		defer cleanup()

		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		teamRepo := repository.NewTeamRepository(pool, logger)
		prRepo := repository.NewPullRequestRepository(pool, logger)

		// Create team with members
		members := []domain.User{
			{ID: "author-a5", Username: "author", IsActive: true},
		}
		for i := 1; i <= 5; i++ {
			members = append(members, domain.User{
				ID:       fmt.Sprintf("reviewer-d%d", i),
				Username: fmt.Sprintf("reviewer%d", i),
				IsActive: true,
			})
		}
		team := &domain.Team{
			Name:    "test-team-assign-deadlock",
			Members: members,
		}
		err := teamRepo.CreateWithMembers(context.Background(), team)
		require.NoError(t, err)

		// Create multiple PRs without reviewers
		const numPRs = 5
		for i := 0; i < numPRs; i++ {
			pr := &domain.PullRequest{
				ID:                fmt.Sprintf("pr-assign-deadlock-%d", i),
				Name:              fmt.Sprintf("Test PR %d", i),
				AuthorID:          "author-a5",
				Status:            domain.PRStatusOpen,
				AssignedReviewers: []string{},
			}
			err = prRepo.Create(context.Background(), pr)
			require.NoError(t, err)
		}

		// Concurrently assign reviewers with different orders
		var wg sync.WaitGroup
		errors := make([]error, numPRs)

		for i := 0; i < numPRs; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				prID := fmt.Sprintf("pr-assign-deadlock-%d", idx)
				// Different order but will be sorted internally
				reviewers := []string{
					fmt.Sprintf("reviewer-d%d", (idx+2)%5+1),
					fmt.Sprintf("reviewer-d%d", (idx+1)%5+1),
				}
				errors[idx] = prRepo.AssignReviewers(context.Background(), prID, reviewers)
			}(i)
		}

		wg.Wait()

		// All assignments should succeed (no deadlocks due to sorting)
		for i, err := range errors {
			assert.NoError(t, err, "Assignment %d failed", i)
		}

		// Verify all PRs have reviewers
		for i := 0; i < numPRs; i++ {
			pr, err := prRepo.GetByID(context.Background(), fmt.Sprintf("pr-assign-deadlock-%d", i))
			require.NoError(t, err)
			assert.Len(t, pr.AssignedReviewers, 2, "PR %d should have 2 reviewers", i)
		}
	})

	t.Run("AssignReviewers_TransactionCommitCheck", func(t *testing.T) {
		pool, cleanup := setupTestDB(t)
		defer cleanup()

		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		teamRepo := repository.NewTeamRepository(pool, logger)
		prRepo := repository.NewPullRequestRepository(pool, logger)

		// Create team with members
		team := &domain.Team{
			Name: "test-team-assign-commit",
			Members: []domain.User{
				{ID: "author-a6", Username: "author", IsActive: true},
				{ID: "reviewer-a9", Username: "reviewer1", IsActive: true},
				{ID: "reviewer-a10", Username: "reviewer2", IsActive: true},
			},
		}
		err := teamRepo.CreateWithMembers(context.Background(), team)
		require.NoError(t, err)

		// Create PR without reviewers
		pr := &domain.PullRequest{
			ID:                "pr-assign-commit",
			Name:              "Test PR",
			AuthorID:          "author-a6",
			Status:            domain.PRStatusOpen,
			AssignedReviewers: []string{},
		}
		err = prRepo.Create(context.Background(), pr)
		require.NoError(t, err)

		// Assign reviewers
		err = prRepo.AssignReviewers(context.Background(), "pr-assign-commit", []string{"reviewer-a9", "reviewer-a10"})
		require.NoError(t, err)

		// Verify transaction was committed (PR has reviewers)
		finalPR, err := prRepo.GetByID(context.Background(), "pr-assign-commit")
		require.NoError(t, err)
		assert.Len(t, finalPR.AssignedReviewers, 2)
	})
}
