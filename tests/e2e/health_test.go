package e2e

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestHealthEndpoint проверяет health check эндпоинт
func TestHealthEndpoint(t *testing.T) {
	client := NewTestClient()
	client.WaitForService(t, 30)

	resp := client.Get(t, "/health")
	defer func() { _ = resp.Body.Close() }()

	AssertStatusCode(t, resp, http.StatusOK)

	var result map[string]interface{}
	client.DecodeJSON(t, resp, &result)

	assert.Equal(t, "ok", result["status"])
}

// TestStatsEndpoint проверяет эндпоинт статистики
func TestStatsEndpoint(t *testing.T) {
	client := NewTestClient()

	resp := client.Get(t, "/stats")
	defer func() { _ = resp.Body.Close() }()

	AssertStatusCode(t, resp, http.StatusOK)

	var stats map[string]interface{}
	client.DecodeJSON(t, resp, &stats)

	// Проверяем наличие всех полей
	assert.Contains(t, stats, "total_prs")
	assert.Contains(t, stats, "open_prs")
	assert.Contains(t, stats, "merged_prs")
	assert.Contains(t, stats, "total_teams")
	assert.Contains(t, stats, "total_users")
	assert.Contains(t, stats, "active_users")
}
