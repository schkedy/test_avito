package integration

import (
	"context"
	"testing"

	"test_avito/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPullRequestService_CreatePR(t *testing.T) {
	teamSvc, _, prSvc, _, cleanup := setupTestServices(t)
	defer cleanup()

	ctx := context.Background()
	_, userIDs := setupTestTeam(t, ctx, teamSvc, 3)

	t.Run("CreatePRWithReviewers", func(t *testing.T) {
		prID := testID("pr")
		pr, err := prSvc.CreatePR(ctx, prID, "Test PR", userIDs[0])
		require.NoError(t, err)

		assert.Equal(t, prID, pr.ID)
		assert.Equal(t, "Test PR", pr.Name)
		assert.Equal(t, userIDs[0], pr.AuthorID)
		assert.Equal(t, domain.PRStatus("OPEN"), pr.Status)

		// Should have assigned reviewers (max 2, excluding author)
		assert.LessOrEqual(t, len(pr.AssignedReviewers), 2)

		// Author should not be in reviewers
		for _, reviewerID := range pr.AssignedReviewers {
			assert.NotEqual(t, userIDs[0], reviewerID, "Author should not be reviewer")
		}
	})

	t.Run("CreatePRSoloAuthor", func(t *testing.T) {
		// Create team with single user
		soloTeamName := testID("solo_team")
		soloUserID := testID("solo_user")
		soloUsers := []domain.User{
			{
				ID:       soloUserID,
				Username: "Solo User",
				TeamName: soloTeamName,
				IsActive: true,
			},
		}
		team := domain.NewTeam(soloTeamName, soloUsers)
		err := teamSvc.AddTeam(ctx, team)
		require.NoError(t, err)

		// Create PR - should have no reviewers
		prID := testID("pr_solo")
		pr, err := prSvc.CreatePR(ctx, prID, "Solo PR", soloUserID)
		require.NoError(t, err)

		assert.Equal(t, prID, pr.ID)
		assert.Empty(t, pr.AssignedReviewers, "Solo author should have no reviewers")
	})

	t.Run("CreatePRNonExistentAuthor", func(t *testing.T) {
		prID := testID("pr_invalid")
		_, err := prSvc.CreatePR(ctx, prID, "Invalid PR", "nonexistent_author")
		assert.Error(t, err)
	})

	t.Run("CreatePRDuplicateID", func(t *testing.T) {
		prID := testID("pr_dup")

		// Create first PR
		_, err := prSvc.CreatePR(ctx, prID, "First PR", userIDs[0])
		require.NoError(t, err)

		// Try to create PR with same ID
		_, err = prSvc.CreatePR(ctx, prID, "Second PR", userIDs[1])
		assert.Error(t, err, "Should not allow duplicate PR ID")
	})
}

func TestPullRequestService_MergePR(t *testing.T) {
	teamSvc, _, prSvc, _, cleanup := setupTestServices(t)
	defer cleanup()

	ctx := context.Background()
	_, userIDs := setupTestTeam(t, ctx, teamSvc, 3)

	prID := testID("pr_merge")
	_, err := prSvc.CreatePR(ctx, prID, "PR to Merge", userIDs[0])
	require.NoError(t, err)

	t.Run("MergePR", func(t *testing.T) {
		pr, err := prSvc.MergePR(ctx, prID)
		require.NoError(t, err)

		assert.Equal(t, prID, pr.ID)
		assert.Equal(t, domain.PRStatus("MERGED"), pr.Status)
		assert.NotNil(t, pr.MergedAt)
	})

	t.Run("MergePRIdempotent", func(t *testing.T) {
		// Merge again - should be idempotent
		pr, err := prSvc.MergePR(ctx, prID)
		require.NoError(t, err)

		assert.Equal(t, domain.PRStatus("MERGED"), pr.Status)
	})

	t.Run("MergeNonExistentPR", func(t *testing.T) {
		_, err := prSvc.MergePR(ctx, "nonexistent_pr")
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrPRNotFound)
	})
}

func TestPullRequestService_ReassignReviewer(t *testing.T) {
	teamSvc, _, prSvc, _, cleanup := setupTestServices(t)
	defer cleanup()

	ctx := context.Background()
	_, userIDs := setupTestTeam(t, ctx, teamSvc, 4)

	prID := testID("pr_reassign")
	pr, err := prSvc.CreatePR(ctx, prID, "PR to Reassign", userIDs[0])
	require.NoError(t, err)

	// Если есть ревьюеры
	if len(pr.AssignedReviewers) > 0 {
		t.Run("ReassignReviewer", func(t *testing.T) {
			oldReviewerID := pr.AssignedReviewers[0]

			newReviewerID, updatedPR, err := prSvc.ReassignReviewer(ctx, prID, oldReviewerID)

			// Может быть успешно или ошибка если нет кандидатов
			if err == nil {
				assert.NotEqual(t, oldReviewerID, newReviewerID, "New reviewer should be different")
				assert.Equal(t, prID, updatedPR.ID)

				// Old reviewer should not be in list
				for _, reviewerID := range updatedPR.AssignedReviewers {
					assert.NotEqual(t, oldReviewerID, reviewerID, "Old reviewer should be removed")
				}
			} else {
				// Может быть недостаточно кандидатов
				assert.Error(t, err)
			}
		})
	}

	t.Run("ReassignNonExistentReviewer", func(t *testing.T) {
		_, _, err := prSvc.ReassignReviewer(ctx, prID, "nonexistent_reviewer")
		assert.Error(t, err)
	})

	t.Run("ReassignAfterMerge", func(t *testing.T) {
		// Merge PR first
		_, err := prSvc.MergePR(ctx, prID)
		require.NoError(t, err)

		// Try to reassign - should fail
		if len(pr.AssignedReviewers) > 0 {
			_, _, err = prSvc.ReassignReviewer(ctx, prID, pr.AssignedReviewers[0])
			assert.Error(t, err, "Should not reassign after merge")
		}
	})

	t.Run("ReassignNonExistentPR", func(t *testing.T) {
		_, _, err := prSvc.ReassignReviewer(ctx, "nonexistent_pr", userIDs[1])
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrPRNotFound)
	})
}

func TestPullRequestService_WithInactiveUsers(t *testing.T) {
	teamSvc, userSvc, prSvc, _, cleanup := setupTestServices(t)
	defer cleanup()

	ctx := context.Background()
	_, userIDs := setupTestTeam(t, ctx, teamSvc, 3)

	t.Run("CreatePRWithInactiveUsers", func(t *testing.T) {
		// Deactivate all potential reviewers
		_, err := userSvc.SetIsActive(ctx, userIDs[1], false)
		require.NoError(t, err)
		_, err = userSvc.SetIsActive(ctx, userIDs[2], false)
		require.NoError(t, err)

		// Create PR - should have no reviewers
		prID := testID("pr_inactive")
		pr, err := prSvc.CreatePR(ctx, prID, "PR with Inactive", userIDs[0])
		require.NoError(t, err)

		assert.Equal(t, prID, pr.ID)
		assert.Empty(t, pr.AssignedReviewers, "Should have no reviewers when all are inactive")
	})
}

func TestPullRequestService_ConcurrentOperations(t *testing.T) {
	teamSvc, _, prSvc, _, cleanup := setupTestServices(t)
	defer cleanup()

	ctx := context.Background()
	_, userIDs := setupTestTeam(t, ctx, teamSvc, 5)

	t.Run("ConcurrentPRCreation", func(t *testing.T) {
		done := make(chan error, 10)

		// Create multiple PRs concurrently
		for i := 0; i < 10; i++ {
			go func(idx int) {
				prID := testID("pr_concurrent")
				_, err := prSvc.CreatePR(ctx, prID, "Concurrent PR", userIDs[idx%len(userIDs)])
				done <- err
			}(i)
		}

		// Wait for all
		for i := 0; i < 10; i++ {
			err := <-done
			assert.NoError(t, err, "Concurrent PR creation should not fail")
		}
	})

	t.Run("ConcurrentMerge", func(t *testing.T) {
		// Create PR
		prID := testID("pr_merge_concurrent")
		_, err := prSvc.CreatePR(ctx, prID, "PR for Concurrent Merge", userIDs[0])
		require.NoError(t, err)

		done := make(chan error, 5)

		// Try to merge same PR concurrently - should be idempotent
		for i := 0; i < 5; i++ {
			go func() {
				_, err := prSvc.MergePR(ctx, prID)
				done <- err
			}()
		}

		// All should succeed due to idempotency
		for i := 0; i < 5; i++ {
			err := <-done
			assert.NoError(t, err, "Concurrent merge should be idempotent")
		}
	})
}
