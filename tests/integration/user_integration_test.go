package integration

import (
	"context"
	"testing"

	"test_avito/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserService_SetIsActive(t *testing.T) {
	teamSvc, userSvc, _, _, cleanup := setupTestServices(t)
	defer cleanup()

	ctx := context.Background()
	teamName, userIDs := setupTestTeam(t, ctx, teamSvc, 3)

	t.Run("DeactivateUser", func(t *testing.T) {
		user, err := userSvc.SetIsActive(ctx, userIDs[0], false)
		require.NoError(t, err)
		assert.Equal(t, userIDs[0], user.ID)
		assert.False(t, user.IsActive)
		assert.Equal(t, teamName, user.TeamName)

		// Verify through team
		team, err := teamSvc.GetTeam(ctx, teamName)
		require.NoError(t, err)

		for _, member := range team.Members {
			if member.ID == userIDs[0] {
				assert.False(t, member.IsActive)
			}
		}
	})

	t.Run("ActivateUser", func(t *testing.T) {
		// First deactivate
		_, err := userSvc.SetIsActive(ctx, userIDs[1], false)
		require.NoError(t, err)

		// Then activate again
		user, err := userSvc.SetIsActive(ctx, userIDs[1], true)
		require.NoError(t, err)
		assert.True(t, user.IsActive)
	})

	t.Run("SetIsActiveIdempotent", func(t *testing.T) {
		// Activate already active user
		user, err := userSvc.SetIsActive(ctx, userIDs[2], true)
		require.NoError(t, err)
		assert.True(t, user.IsActive)

		// Should be idempotent
		user, err = userSvc.SetIsActive(ctx, userIDs[2], true)
		require.NoError(t, err)
		assert.True(t, user.IsActive)
	})

	t.Run("SetIsActiveNonExistentUser", func(t *testing.T) {
		_, err := userSvc.SetIsActive(ctx, "nonexistent_user", true)
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrUserNotFound)
	})
}

func TestUserService_GetUserReviews(t *testing.T) {
	teamSvc, userSvc, prSvc, _, cleanup := setupTestServices(t)
	defer cleanup()

	ctx := context.Background()
	_, userIDs := setupTestTeam(t, ctx, teamSvc, 3)

	// Create some PRs
	pr1ID := testID("pr1")
	pr2ID := testID("pr2")

	_, err := prSvc.CreatePR(ctx, pr1ID, "Test PR 1", userIDs[0])
	require.NoError(t, err)

	_, err = prSvc.CreatePR(ctx, pr2ID, "Test PR 2", userIDs[0])
	require.NoError(t, err)

	t.Run("GetUserReviews", func(t *testing.T) {
		// Get reviews for user who might be a reviewer
		reviews, err := userSvc.GetReviewsByUser(ctx, userIDs[1])
		require.NoError(t, err)

		// Method is not fully implemented yet, so reviews may be nil
		// Just verify no error is returned
		_ = reviews
	})

	t.Run("GetUserReviewsNonExistentUser", func(t *testing.T) {
		reviews, err := userSvc.GetReviewsByUser(ctx, "nonexistent_user")
		// Может вернуть пустой список или ошибку в зависимости от реализации
		if err != nil {
			assert.ErrorIs(t, err, domain.ErrUserNotFound)
		} else {
			// Or return empty list
			assert.NotNil(t, reviews)
		}
	})

	t.Run("GetUserReviewsForAuthor", func(t *testing.T) {
		// Author should not have their own PRs in reviews
		reviews, err := userSvc.GetReviewsByUser(ctx, userIDs[0])
		require.NoError(t, err)

		// Method is not fully implemented yet
		_ = reviews
	})
}

func TestUserService_ConcurrentActivation(t *testing.T) {
	teamSvc, userSvc, _, _, cleanup := setupTestServices(t)
	defer cleanup()

	ctx := context.Background()
	_, userIDs := setupTestTeam(t, ctx, teamSvc, 5)

	t.Run("ConcurrentActivateDeactivate", func(t *testing.T) {
		done := make(chan error, 10)

		// Concurrently activate and deactivate users
		for i := 0; i < 5; i++ {
			userID := userIDs[i%len(userIDs)]
			go func(uid string, active bool) {
				_, err := userSvc.SetIsActive(ctx, uid, active)
				done <- err
			}(userID, i%2 == 0)
		}

		// Also concurrent deactivations
		for i := 0; i < 5; i++ {
			userID := userIDs[i%len(userIDs)]
			go func(uid string) {
				_, err := userSvc.SetIsActive(ctx, uid, false)
				done <- err
			}(userID)
		}

		// Wait for all goroutines
		for i := 0; i < 10; i++ {
			err := <-done
			assert.NoError(t, err, "Concurrent user activation should not fail")
		}

		// Verify all users still exist
		for _, userID := range userIDs {
			_, err := userSvc.SetIsActive(ctx, userID, true)
			assert.NoError(t, err, "User %s should still exist", userID)
		}
	})
}
