package integration

import (
	"context"
	"testing"

	"test_avito/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTeamService_CreateTeam(t *testing.T) {
	teamSvc, _, _, _, cleanup := setupTestServices(t)
	defer cleanup()

	ctx := context.Background()
	teamName := testID("team")

	members := []domain.User{
		{
			ID:       testID("user1"),
			Username: "Alice",
			TeamName: teamName,
			IsActive: true,
		},
		{
			ID:       testID("user2"),
			Username: "Bob",
			TeamName: teamName,
			IsActive: true,
		},
	}

	t.Run("CreateNewTeam", func(t *testing.T) {
		team := domain.NewTeam(teamName, members)
		err := teamSvc.AddTeam(ctx, team)
		require.NoError(t, err)

		// Verify team was created
		gotTeam, err := teamSvc.GetTeam(ctx, teamName)
		require.NoError(t, err)
		assert.Equal(t, teamName, gotTeam.Name)
		assert.Len(t, gotTeam.Members, 2)
	})
}

func TestTeamService_GetTeam(t *testing.T) {
	teamSvc, _, _, _, cleanup := setupTestServices(t)
	defer cleanup()

	ctx := context.Background()
	teamName, userIDs := setupTestTeam(t, ctx, teamSvc, 3)

	t.Run("GetExistingTeam", func(t *testing.T) {
		team, err := teamSvc.GetTeam(ctx, teamName)
		require.NoError(t, err)
		assert.Equal(t, teamName, team.Name)
		assert.Len(t, team.Members, 3)

		// Verify all users are present
		userIDsMap := make(map[string]bool)
		for _, member := range team.Members {
			userIDsMap[member.ID] = true
		}

		for _, userID := range userIDs {
			assert.True(t, userIDsMap[userID], "User %s should be in team", userID)
		}
	})

	t.Run("GetNonExistentTeam", func(t *testing.T) {
		_, err := teamSvc.GetTeam(ctx, "nonexistent_team")
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrTeamNotFound)
	})
}

func TestTeamService_UpdateTeam(t *testing.T) {
	teamSvc, _, _, _, cleanup := setupTestServices(t)
	defer cleanup()

	ctx := context.Background()
	teamName, userIDs := setupTestTeam(t, ctx, teamSvc, 2)

	t.Run("AddNewMembers", func(t *testing.T) {
		newUserID := testID("user_new")
		updatedMembers := []domain.User{
			{
				ID:       userIDs[0],
				Username: "Alice Updated",
				TeamName: teamName,
				IsActive: true,
			},
			{
				ID:       newUserID,
				Username: "Charlie",
				TeamName: teamName,
				IsActive: true,
			},
		}

		team := domain.NewTeam(teamName, updatedMembers)
		err := teamSvc.AddTeam(ctx, team)
		require.NoError(t, err)

		// Verify team has new member
		gotTeam, err := teamSvc.GetTeam(ctx, teamName)
		require.NoError(t, err)

		// Should have at least the new user
		userIDsMap := make(map[string]bool)
		for _, member := range gotTeam.Members {
			userIDsMap[member.ID] = true
		}
		assert.True(t, userIDsMap[newUserID], "New user should be in team")
	})

	t.Run("UpdateMemberInfo", func(t *testing.T) {
		updatedMembers := []domain.User{
			{
				ID:       userIDs[0],
				Username: "Alice SUPER Updated",
				TeamName: teamName,
				IsActive: false, // Deactivate
			},
		}

		team := domain.NewTeam(teamName, updatedMembers)
		err := teamSvc.AddTeam(ctx, team)
		require.NoError(t, err)

		// Verify user was updated
		gotTeam, err := teamSvc.GetTeam(ctx, teamName)
		require.NoError(t, err)

		for _, member := range gotTeam.Members {
			if member.ID == userIDs[0] {
				assert.Equal(t, "Alice SUPER Updated", member.Username)
				assert.False(t, member.IsActive)
			}
		}
	})
}

func TestTeamService_DeactivateTeam(t *testing.T) {
	teamSvc, _, _, _, cleanup := setupTestServices(t)
	defer cleanup()

	ctx := context.Background()
	teamName, _ := setupTestTeam(t, ctx, teamSvc, 3)

	t.Run("DeactivateAllMembers", func(t *testing.T) {
		deactivatedTeam, deactivatedCount, err := teamSvc.DeactivateTeam(ctx, teamName)
		require.NoError(t, err)
		assert.Equal(t, 3, deactivatedCount)
		assert.Equal(t, teamName, deactivatedTeam.Name)

		// Verify all members are inactive
		team, err := teamSvc.GetTeam(ctx, teamName)
		require.NoError(t, err)

		for _, member := range team.Members {
			assert.False(t, member.IsActive, "Member %s should be inactive", member.ID)
		}
	})

	t.Run("DeactivateNonExistentTeam", func(t *testing.T) {
		_, _, err := teamSvc.DeactivateTeam(ctx, "nonexistent_team")
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrTeamNotFound)
	})

	t.Run("DeactivateAlreadyDeactivated", func(t *testing.T) {
		// Deactivate again - should be idempotent
		_, deactivatedCount, err := teamSvc.DeactivateTeam(ctx, teamName)
		require.NoError(t, err)
		assert.Equal(t, 0, deactivatedCount, "No users should be deactivated second time")
	})
}

func TestTeamService_ConcurrentAccess(t *testing.T) {
	teamSvc, _, _, _, cleanup := setupTestServices(t)
	defer cleanup()

	ctx := context.Background()
	teamName := testID("team_concurrent")

	t.Run("ConcurrentTeamCreation", func(t *testing.T) {
		// Create team concurrently from multiple goroutines
		done := make(chan error, 5)

		for i := 0; i < 5; i++ {
			go func() {
				members := []domain.User{
					{
						ID:       testID("user_concurrent"),
						Username: "User Concurrent",
						TeamName: teamName,
						IsActive: true,
					},
				}
				team := domain.NewTeam(teamName, members)
				done <- teamSvc.AddTeam(ctx, team)
			}()
		}

		// Wait for all goroutines
		for i := 0; i < 5; i++ {
			err := <-done
			assert.NoError(t, err, "Concurrent team creation should not fail")
		}

		// Verify team exists
		team, err := teamSvc.GetTeam(ctx, teamName)
		require.NoError(t, err)
		assert.Equal(t, teamName, team.Name)
	})
}
