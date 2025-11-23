package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatsService_GetStats(t *testing.T) {
	teamSvc, userSvc, prSvc, statsSvc, cleanup := setupTestServices(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("EmptyStats", func(t *testing.T) {
		stats, err := statsSvc.GetStats(ctx)
		require.NoError(t, err)

		assert.Equal(t, 0, stats.TotalPRs)
		assert.Equal(t, 0, stats.OpenPRs)
		assert.Equal(t, 0, stats.MergedPRs)
		assert.Equal(t, 0, stats.TotalTeams)
		assert.Equal(t, 0, stats.TotalUsers)
		assert.Equal(t, 0, stats.ActiveUsers)
	})

	t.Run("StatsWithData", func(t *testing.T) {
		// Create teams
		_, userIDs1 := setupTestTeam(t, ctx, teamSvc, 3)
		_, userIDs2 := setupTestTeam(t, ctx, teamSvc, 2)

		// Create some PRs
		pr1ID := testID("pr_stats1")
		pr2ID := testID("pr_stats2")
		pr3ID := testID("pr_stats3")

		_, err := prSvc.CreatePR(ctx, pr1ID, "PR 1", userIDs1[0])
		require.NoError(t, err)

		_, err = prSvc.CreatePR(ctx, pr2ID, "PR 2", userIDs1[1])
		require.NoError(t, err)

		_, err = prSvc.CreatePR(ctx, pr3ID, "PR 3", userIDs2[0])
		require.NoError(t, err)

		// Merge one PR
		_, err = prSvc.MergePR(ctx, pr1ID)
		require.NoError(t, err)

		// Get stats
		stats, err := statsSvc.GetStats(ctx)
		require.NoError(t, err)

		assert.GreaterOrEqual(t, stats.TotalPRs, 3, "Should have at least 3 PRs")
		assert.GreaterOrEqual(t, stats.OpenPRs, 2, "Should have at least 2 open PRs")
		assert.GreaterOrEqual(t, stats.MergedPRs, 1, "Should have at least 1 merged PR")
		assert.GreaterOrEqual(t, stats.TotalTeams, 2, "Should have at least 2 teams")
		assert.GreaterOrEqual(t, stats.TotalUsers, 5, "Should have at least 5 users")
		assert.GreaterOrEqual(t, stats.ActiveUsers, 5, "All users should be active")
	})

	t.Run("StatsWithInactiveUsers", func(t *testing.T) {
		// Create team
		_, userIDs := setupTestTeam(t, ctx, teamSvc, 4)

		// Deactivate some users
		_, err := userSvc.SetIsActive(ctx, userIDs[0], false)
		require.NoError(t, err)

		_, err = userSvc.SetIsActive(ctx, userIDs[1], false)
		require.NoError(t, err)

		// Get stats
		stats, err := statsSvc.GetStats(ctx)
		require.NoError(t, err)

		assert.GreaterOrEqual(t, stats.TotalUsers, 4, "Should count all users")
		assert.LessOrEqual(t, stats.ActiveUsers, stats.TotalUsers, "Active should be <= total")
	})

	t.Run("StatsAfterDeactivateTeam", func(t *testing.T) {
		// Create team
		teamName, _ := setupTestTeam(t, ctx, teamSvc, 3)

		// Get initial active count
		stats1, err := statsSvc.GetStats(ctx)
		require.NoError(t, err)
		initialActive := stats1.ActiveUsers

		// Deactivate team
		_, _, err = teamSvc.DeactivateTeam(ctx, teamName)
		require.NoError(t, err)

		// Get updated stats
		stats2, err := statsSvc.GetStats(ctx)
		require.NoError(t, err)

		assert.LessOrEqual(t, stats2.ActiveUsers, initialActive, "Active users should decrease")
	})

	t.Run("StatsAccuracy", func(t *testing.T) {
		// Verify total PRs = open + merged
		stats, err := statsSvc.GetStats(ctx)
		require.NoError(t, err)

		assert.Equal(t, stats.TotalPRs, stats.OpenPRs+stats.MergedPRs,
			"Total PRs should equal Open + Merged")

		assert.LessOrEqual(t, stats.ActiveUsers, stats.TotalUsers,
			"Active users cannot exceed total users")
	})
}

func TestStatsService_Consistency(t *testing.T) {
	teamSvc, _, prSvc, statsSvc, cleanup := setupTestServices(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("ConsistentStatsAfterOperations", func(t *testing.T) {
		// Get initial stats
		stats1, err := statsSvc.GetStats(ctx)
		require.NoError(t, err)

		// Create team and PR
		_, userIDs := setupTestTeam(t, ctx, teamSvc, 2)
		prID := testID("pr_consistency")
		_, err = prSvc.CreatePR(ctx, prID, "Consistency PR", userIDs[0])
		require.NoError(t, err)

		// Get updated stats
		stats2, err := statsSvc.GetStats(ctx)
		require.NoError(t, err)

		// Verify changes
		assert.Greater(t, stats2.TotalPRs, stats1.TotalPRs, "Total PRs should increase")
		assert.Greater(t, stats2.OpenPRs, stats1.OpenPRs, "Open PRs should increase")
		assert.Greater(t, stats2.TotalUsers, stats1.TotalUsers, "Total users should increase")

		// Merge PR
		_, err = prSvc.MergePR(ctx, prID)
		require.NoError(t, err)

		// Get final stats
		stats3, err := statsSvc.GetStats(ctx)
		require.NoError(t, err)

		// Verify PR counts updated correctly
		assert.Equal(t, stats2.TotalPRs, stats3.TotalPRs, "Total PRs should stay same")
		assert.Less(t, stats3.OpenPRs, stats2.OpenPRs, "Open PRs should decrease")
		assert.Greater(t, stats3.MergedPRs, stats2.MergedPRs, "Merged PRs should increase")
	})
}
