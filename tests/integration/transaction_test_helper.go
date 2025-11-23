package integration

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

// Test database connection string
const testDSN = "postgres://postgres:postgres@localhost:5434/postgres?sslmode=disable"

func setupTestDB(t *testing.T) (*pgxpool.Pool, func()) {
	t.Helper()

	pool, err := pgxpool.New(context.Background(), testDSN)
	require.NoError(t, err, "Failed to connect to test database")

	// Cleanup function
	cleanup := func() {
		cleanupTestData(t, pool)
		pool.Close()
	}

	return pool, cleanup
}

func cleanupTestData(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()

	// Delete in correct order due to foreign keys
	_, _ = pool.Exec(ctx, "DELETE FROM reviewers")
	_, _ = pool.Exec(ctx, "DELETE FROM pull_requests")
	_, _ = pool.Exec(ctx, "DELETE FROM users")
	_, _ = pool.Exec(ctx, "DELETE FROM teams")
}
