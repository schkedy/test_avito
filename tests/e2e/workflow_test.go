package e2e

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestCompleteWorkflow тестирует полный workflow от создания команды до merge PR
func TestCompleteWorkflow(t *testing.T) {
	client := NewTestClient()
	client.WaitForService(t, 30)

	timestamp := time.Now().Unix()
	teamName := fmt.Sprintf("e2e_workflow_team_%d", timestamp)
	authorID := fmt.Sprintf("wf_author_%d", timestamp)
	reviewer1ID := fmt.Sprintf("wf_rev1_%d", timestamp)
	reviewer2ID := fmt.Sprintf("wf_rev2_%d", timestamp)
	reviewer3ID := fmt.Sprintf("wf_rev3_%d", timestamp)
	prID1 := fmt.Sprintf("wf_pr1_%d", timestamp)
	prID2 := fmt.Sprintf("wf_pr2_%d", timestamp)

	t.Run("CreateTeam", func(t *testing.T) {
		payload := map[string]interface{}{
			"team_name": teamName,
			"members": []map[string]interface{}{
				{
					"user_id":   authorID,
					"username":  "Main Author",
					"is_active": true,
				},
				{
					"user_id":   reviewer1ID,
					"username":  "Reviewer One",
					"is_active": true,
				},
				{
					"user_id":   reviewer2ID,
					"username":  "Reviewer Two",
					"is_active": true,
				},
			},
		}

		resp := client.Post(t, "/team/add", payload)
		defer func() { _ = resp.Body.Close() }()

		AssertStatusCode(t, resp, http.StatusCreated)

		var result map[string]interface{}
		client.DecodeJSON(t, resp, &result)

		assert.Contains(t, result, "team")
		team := result["team"].(map[string]interface{})
		assert.Equal(t, teamName, team["team_name"])
	})

	t.Run("GetTeam", func(t *testing.T) {
		resp := client.Get(t, fmt.Sprintf("/team/get?team_name=%s", teamName))
		defer func() { _ = resp.Body.Close() }()

		AssertStatusCode(t, resp, http.StatusOK)

		var result map[string]interface{}
		client.DecodeJSON(t, resp, &result)

		assert.Equal(t, teamName, result["team_name"])
		members := result["members"].([]interface{})
		assert.Len(t, members, 3)
	})

	t.Run("CreateFirstPR", func(t *testing.T) {
		payload := map[string]interface{}{
			"pull_request_id":   prID1,
			"pull_request_name": fmt.Sprintf("Feature A %d", timestamp),
			"author_id":         authorID,
		}

		resp := client.Post(t, "/pullRequest/create", payload)
		defer func() { _ = resp.Body.Close() }()

		AssertStatusCode(t, resp, http.StatusCreated)

		var result map[string]interface{}
		client.DecodeJSON(t, resp, &result)

		pr := result["pr"].(map[string]interface{})
		if reviewers, ok := pr["assigned_reviewers"]; ok && reviewers != nil {
			reviewerList := reviewers.([]interface{})
			assert.LessOrEqual(t, len(reviewerList), 2)
		}
	})

	t.Run("CreateSecondPR", func(t *testing.T) {
		payload := map[string]interface{}{
			"pull_request_id":   prID2,
			"pull_request_name": fmt.Sprintf("Feature B %d", timestamp),
			"author_id":         authorID,
		}

		resp := client.Post(t, "/pullRequest/create", payload)
		defer func() { _ = resp.Body.Close() }()

		AssertStatusCode(t, resp, http.StatusCreated)
	})

	t.Run("CheckStats", func(t *testing.T) {
		resp := client.Get(t, "/stats")
		defer func() { _ = resp.Body.Close() }()

		AssertStatusCode(t, resp, http.StatusOK)

		var result map[string]interface{}
		client.DecodeJSON(t, resp, &result)

		assert.Contains(t, result, "total_prs")
		assert.Contains(t, result, "open_prs")
		assert.Contains(t, result, "total_teams")
		assert.Contains(t, result, "total_users")
	})

	t.Run("MergeFirstPR", func(t *testing.T) {
		payload := map[string]interface{}{
			"pull_request_id": prID1,
		}

		resp := client.Post(t, "/pullRequest/merge", payload)
		defer func() { _ = resp.Body.Close() }()

		AssertStatusCode(t, resp, http.StatusOK)

		var result map[string]interface{}
		client.DecodeJSON(t, resp, &result)

		pr := result["pr"].(map[string]interface{})
		assert.Equal(t, "MERGED", pr["status"])
	})

	t.Run("AddTeamMember", func(t *testing.T) {
		payload := map[string]interface{}{
			"team_name": teamName,
			"members": []map[string]interface{}{
				{
					"user_id":   reviewer3ID,
					"username":  "Reviewer Three",
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
	})

	t.Run("CheckUpdatedTeam", func(t *testing.T) {
		resp := client.Get(t, fmt.Sprintf("/team/get?team_name=%s", teamName))
		defer func() { _ = resp.Body.Close() }()

		AssertStatusCode(t, resp, http.StatusOK)

		var result map[string]interface{}
		client.DecodeJSON(t, resp, &result)

		members := result["members"].([]interface{})
		assert.GreaterOrEqual(t, len(members), 3)
	})

	t.Run("DeactivateReviewer", func(t *testing.T) {
		payload := map[string]interface{}{
			"user_id":   reviewer1ID,
			"is_active": false,
		}

		resp := client.Post(t, "/users/setIsActive", payload)
		defer func() { _ = resp.Body.Close() }()
		AssertStatusCode(t, resp, http.StatusOK)
	})

	t.Run("MergeSecondPR", func(t *testing.T) {
		payload := map[string]interface{}{
			"pull_request_id": prID2,
		}

		resp := client.Post(t, "/pullRequest/merge", payload)
		defer func() { _ = resp.Body.Close() }()

		AssertStatusCode(t, resp, http.StatusOK)
	})

	t.Run("FinalStats", func(t *testing.T) {
		resp := client.Get(t, "/stats")
		defer func() { _ = resp.Body.Close() }()

		AssertStatusCode(t, resp, http.StatusOK)

		var result map[string]interface{}
		client.DecodeJSON(t, resp, &result)

		assert.Contains(t, result, "merged_prs")
		assert.Contains(t, result, "total_users")
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
	})
}

// TestErrorHandling тестирует обработку различных ошибок
func TestErrorHandling(t *testing.T) {
	client := NewTestClient()

	t.Run("InvalidTeamName", func(t *testing.T) {
		resp := client.Get(t, "/team/get?team_name=nonexistent_team_99999")
		defer func() { _ = resp.Body.Close() }()
		AssertStatusCode(t, resp, http.StatusNotFound)
	})

	t.Run("InvalidPRID", func(t *testing.T) {
		payload := map[string]interface{}{
			"pull_request_id": "nonexistent_pr_99999",
		}

		resp := client.Post(t, "/pullRequest/merge", payload)
		defer func() { _ = resp.Body.Close() }()
		AssertStatusCode(t, resp, http.StatusNotFound)
	})

	t.Run("MissingRequiredFields", func(t *testing.T) {
		payload := map[string]interface{}{
			"team_name": "incomplete_team",
			// members отсутствует
		}

		resp := client.Post(t, "/team/add", payload)
		defer func() { _ = resp.Body.Close() }()
		assert.NotEqual(t, http.StatusOK, resp.StatusCode)
		assert.NotEqual(t, http.StatusCreated, resp.StatusCode)
	})

	t.Run("InvalidAuthorID", func(t *testing.T) {
		timestamp := time.Now().Unix()
		payload := map[string]interface{}{
			"pull_request_id":   fmt.Sprintf("invalid_pr_%d", timestamp),
			"pull_request_name": "Invalid PR",
			"author_id":         "nonexistent_author_999",
		}

		resp := client.Post(t, "/pullRequest/create", payload)
		defer func() { _ = resp.Body.Close() }()
		// Ожидаем либо 400 либо 404
		assert.True(t, resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusNotFound)
	})
}
