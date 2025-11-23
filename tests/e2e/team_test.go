package e2e

import (
	"fmt"
	"net/http"
	"testing"

	"time"

	"github.com/stretchr/testify/assert"
)

// TestTeamLifecycle тестирует полный жизненный цикл команды
func TestTeamLifecycle(t *testing.T) {
	client := NewTestClient()
	client.WaitForService(t, 30)

	teamName := fmt.Sprintf("e2e_test_team_%d", getTimestamp())

	t.Run("CreateTeam", func(t *testing.T) {
		payload := map[string]interface{}{
			"team_name": teamName,
			"members": []map[string]interface{}{
				{
					"user_id":   fmt.Sprintf("e2e_%d_1001", getTimestamp()),
					"username":  "E2E User 1",
					"is_active": true,
				},
				{
					"user_id":   fmt.Sprintf("e2e_%d_1002", getTimestamp()),
					"username":  "E2E User 2",
					"is_active": true,
				},
			},
		}

		resp := client.Post(t, "/team/add", payload)
		defer func() { _ = resp.Body.Close() }()

		// Может быть 200 или 201
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
			t.Fatalf("Expected 200 or 201, got %d", resp.StatusCode)
		}

		var result map[string]interface{}
		client.DecodeJSON(t, resp, &result)

		assert.Contains(t, result, "team")
		team := result["team"].(map[string]interface{})
		assert.Equal(t, teamName, team["team_name"])

		if members, ok := team["members"]; ok && members != nil {
			memberList := members.([]interface{})
			assert.Len(t, memberList, 2)
		}
	})

	t.Run("GetTeam", func(t *testing.T) {
		resp := client.Get(t, fmt.Sprintf("/team/get?team_name=%s", teamName))
		defer func() { _ = resp.Body.Close() }()

		AssertStatusCode(t, resp, http.StatusOK)

		var team map[string]interface{}
		client.DecodeJSON(t, resp, &team)

		assert.Equal(t, teamName, team["team_name"])
		members := team["members"].([]interface{})
		assert.Len(t, members, 2)
	})

	t.Run("UpdateTeam", func(t *testing.T) {
		payload := map[string]interface{}{
			"team_name": teamName,
			"members": []map[string]interface{}{
				{
					"user_id":   fmt.Sprintf("e2e_%d_1001", getTimestamp()),
					"username":  "E2E User 1 Updated",
					"is_active": true,
				},
				{
					"user_id":   fmt.Sprintf("e2e_%d_1003", getTimestamp()),
					"username":  "E2E User 3",
					"is_active": true,
				},
			},
		}

		resp := client.Post(t, "/team/add", payload)
		defer func() { _ = resp.Body.Close() }()

		// Может быть 200 (updated) или 201 (created)
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
			t.Fatalf("Expected 200 or 201, got %d", resp.StatusCode)
		}
	})

	t.Run("DeactivateTeam", func(t *testing.T) {
		payload := map[string]interface{}{
			"team_name": teamName,
		}

		resp := client.Post(t, "/team/deactivate", payload)
		defer func() { _ = resp.Body.Close() }()

		AssertStatusCode(t, resp, http.StatusOK)

		var result map[string]interface{}
		client.DecodeJSON(t, resp, &result)

		assert.Contains(t, result, "deactivated_count")
		assert.Greater(t, int(result["deactivated_count"].(float64)), 0)
	})
}

// TestTeamNotFound тестирует получение несуществующей команды
func TestTeamNotFound(t *testing.T) {
	client := NewTestClient()

	resp := client.Get(t, "/team/get?team_name=nonexistent_team_12345")
	defer func() { _ = resp.Body.Close() }()

	AssertStatusCode(t, resp, http.StatusNotFound)
}

func getTimestamp() int64 {
	return time.Now().Unix()
}
