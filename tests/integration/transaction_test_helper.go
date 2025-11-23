package integration

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"test_avito/internal/domain"
	"test_avito/internal/repository"
	"test_avito/internal/service"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

// Test database connection string - uses POSTGRES_EXTERNAL_PORT (5434)
func getTestDSN() string {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5434/pr_reviewer?sslmode=disable"
	}
	return dsn
}

// setupTestDB creates a new database connection pool for testing
func setupTestDB(t *testing.T) (*pgxpool.Pool, func()) {
	t.Helper()

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, getTestDSN())
	require.NoError(t, err, "Failed to connect to test database")

	// Verify connection
	err = pool.Ping(ctx)
	require.NoError(t, err, "Failed to ping test database")

	// Cleanup function
	cleanup := func() {
		cleanupTestData(t, pool)
		pool.Close()
	}

	// Clean up before test
	cleanupTestData(t, pool)

	return pool, cleanup
}

// setupTestServices creates all services with test database
func setupTestServices(t *testing.T) (*service.TeamService, *service.UserService, *service.PullRequestService, *service.StatsService, func()) {
	t.Helper()

	pool, cleanup := setupTestDB(t)

	// Create logger for tests (discard output)
	testLogger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelError, // Only log errors in tests
	}))

	// Create repositories
	teamRepo := repository.NewTeamRepository(pool, testLogger)
	userRepo := repository.NewUserRepository(pool, testLogger)
	prRepo := repository.NewPullRequestRepository(pool, testLogger)
	statsRepo := repository.NewStatsRepository(pool, testLogger)

	// Create services
	teamService := service.NewTeamService(teamRepo, userRepo, testLogger)
	userService := service.NewUserService(userRepo, testLogger)
	prService := service.NewPullRequestService(prRepo, userRepo, testLogger)
	statsService := service.NewStatsService(statsRepo, testLogger)

	return teamService, userService, prService, statsService, cleanup
}

// cleanupTestData deletes all test data in correct order
func cleanupTestData(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()

	// Delete in correct order due to foreign keys
	_, _ = pool.Exec(ctx, "DELETE FROM reviewers")
	_, _ = pool.Exec(ctx, "DELETE FROM pull_requests")
	_, _ = pool.Exec(ctx, "DELETE FROM users")
	_, _ = pool.Exec(ctx, "DELETE FROM teams")
}

// Helper to create unique test IDs
func testID(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}

// Helper to setup a test team with users
func setupTestTeam(t *testing.T, ctx context.Context, teamSvc *service.TeamService, userCount int) (string, []string) {
	t.Helper()

	teamName := testID("team")
	userIDs := make([]string, userCount)
	users := make([]domain.User, userCount)

	for i := 0; i < userCount; i++ {
		userIDs[i] = testID(fmt.Sprintf("user%d", i))
		users[i] = domain.User{
			ID:       userIDs[i],
			Username: fmt.Sprintf("User %d", i),
			TeamName: teamName,
			IsActive: true,
		}
	}

	team := domain.NewTeam(teamName, users)
	err := teamSvc.AddTeam(ctx, team)
	require.NoError(t, err)

	return teamName, userIDs
}
