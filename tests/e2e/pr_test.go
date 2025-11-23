package e2e

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestPullRequestLifecycle тестирует полный жизненный цикл Pull Request
func TestPullRequestLifecycle(t *testing.T) {
	client := NewTestClient()
	client.WaitForService(t, 30)

	timestamp := time.Now().Unix()
	teamName := fmt.Sprintf("e2e_pr_team_%d", timestamp)
	authorID := fmt.Sprintf("pr_author_%d", timestamp)
	reviewer1ID := fmt.Sprintf("pr_rev1_%d", timestamp)
	reviewer2ID := fmt.Sprintf("pr_rev2_%d", timestamp)
	prID := fmt.Sprintf("pr_%d", timestamp)

	t.Run("SetupTeam", func(t *testing.T) {
		payload := map[string]interface{}{
			"team_name": teamName,
			"members": []map[string]interface{}{
				{
					"user_id":   authorID,
					"username":  "PR Author",
					"is_active": true,
				},
				{
					"user_id":   reviewer1ID,
					"username":  "Reviewer 1",
					"is_active": true,
				},
				{
					"user_id":   reviewer2ID,
					"username":  "Reviewer 2",
					"is_active": true,
				},
			},
		}

		resp := client.Post(t, "/team/add", payload)
		defer func() { _ = resp.Body.Close() }()
		AssertStatusCode(t, resp, http.StatusCreated)
	})

	t.Run("CreatePR", func(t *testing.T) {
		payload := map[string]interface{}{
			"pull_request_id":   prID,
			"pull_request_name": fmt.Sprintf("E2E Test PR %d", timestamp),
			"author_id":         authorID,
		}

		resp := client.Post(t, "/pullRequest/create", payload)
		defer func() { _ = resp.Body.Close() }()

		AssertStatusCode(t, resp, http.StatusCreated)

		var result map[string]interface{}
		client.DecodeJSON(t, resp, &result)

		assert.Contains(t, result, "pr")
		pr := result["pr"].(map[string]interface{})
		assert.Equal(t, prID, pr["pull_request_id"])

		if reviewers, ok := pr["assigned_reviewers"]; ok && reviewers != nil {
			reviewerList := reviewers.([]interface{})
			assert.LessOrEqual(t, len(reviewerList), 2, "Should have at most 2 reviewers")
		}
	})

	t.Run("GetUserReviews", func(t *testing.T) {
		// Проверяем PR для одного из ревьюеров
		resp := client.Get(t, fmt.Sprintf("/users/getReview?user_id=%s", reviewer1ID))
		defer func() { _ = resp.Body.Close() }()

		AssertStatusCode(t, resp, http.StatusOK)

		var result map[string]interface{}
		client.DecodeJSON(t, resp, &result)

		assert.Contains(t, result, "user_id")
	})

	t.Run("ReassignReviewer", func(t *testing.T) {
		payload := map[string]interface{}{
			"pull_request_id": prID,
			"old_user_id":     reviewer1ID,
		}

		resp := client.Post(t, "/pullRequest/reassign", payload)
		defer func() { _ = resp.Body.Close() }()

		// Может быть 200 или 400 в зависимости от того, был ли reviewer1 назначен
		if resp.StatusCode == http.StatusOK {
			var result map[string]interface{}
			client.DecodeJSON(t, resp, &result)
			assert.Contains(t, result, "pr")
			assert.Contains(t, result, "replaced_by")
		}
	})

	t.Run("MergePR", func(t *testing.T) {
		payload := map[string]interface{}{
			"pull_request_id": prID,
		}

		resp := client.Post(t, "/pullRequest/merge", payload)
		defer func() { _ = resp.Body.Close() }()

		AssertStatusCode(t, resp, http.StatusOK)

		var result map[string]interface{}
		client.DecodeJSON(t, resp, &result)

		assert.Contains(t, result, "pr")
		pr := result["pr"].(map[string]interface{})
		assert.Equal(t, prID, pr["pull_request_id"])
		assert.Equal(t, "MERGED", pr["status"])
	})

	t.Run("MergePRIdempotent", func(t *testing.T) {
		// Повторный merge должен быть идемпотентным
		payload := map[string]interface{}{
			"pull_request_id": prID,
		}

		resp := client.Post(t, "/pullRequest/merge", payload)
		defer func() { _ = resp.Body.Close() }()

		AssertStatusCode(t, resp, http.StatusOK)
	})

	t.Run("ReassignAfterMerge", func(t *testing.T) {
		// Переназначение после merge должно быть запрещено
		payload := map[string]interface{}{
			"pull_request_id": prID,
			"old_user_id":     reviewer1ID,
		}

		resp := client.Post(t, "/pullRequest/reassign", payload)
		defer func() { _ = resp.Body.Close() }()

		// 409 Conflict - правильный статус для конфликта состояния (PR уже merged)
		AssertStatusCode(t, resp, http.StatusConflict)
	})
}

// TestPRWithoutReviewers тестирует создание PR когда нет активных ревьюеров
func TestPRWithoutReviewers(t *testing.T) {
	client := NewTestClient()

	timestamp := time.Now().Unix()
	teamName := fmt.Sprintf("e2e_solo_team_%d", timestamp)
	soloUserID := fmt.Sprintf("solo_user_%d", timestamp)
	soloPRID := fmt.Sprintf("solo_pr_%d", timestamp)

	// Создаем команду с одним участником
	payload := map[string]interface{}{
		"team_name": teamName,
		"members": []map[string]interface{}{
			{
				"user_id":   soloUserID,
				"username":  "Solo Developer",
				"is_active": true,
			},
		},
	}

	resp := client.Post(t, "/team/add", payload)
	defer func() { _ = resp.Body.Close() }()
	AssertStatusCode(t, resp, http.StatusCreated)

	// Создаем PR - автор один в команде, ревьюеров быть не должно
	prPayload := map[string]interface{}{
		"pull_request_id":   soloPRID,
		"pull_request_name": fmt.Sprintf("Solo PR %d", timestamp),
		"author_id":         soloUserID,
	}

	prResp := client.Post(t, "/pullRequest/create", prPayload)
	defer func() { _ = prResp.Body.Close() }()

	AssertStatusCode(t, prResp, http.StatusCreated)

	var result map[string]interface{}
	client.DecodeJSON(t, prResp, &result)

	pr := result["pr"].(map[string]interface{})
	if reviewers, ok := pr["assigned_reviewers"]; ok && reviewers != nil {
		reviewerList := reviewers.([]interface{})
		assert.Empty(t, reviewerList, "Solo author should have no reviewers")
	}
}
