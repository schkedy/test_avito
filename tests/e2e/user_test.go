package e2e

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestUserActivation тестирует активацию и деактивацию пользователей
func TestUserActivation(t *testing.T) {
	client := NewTestClient()
	client.WaitForService(t, 30)

	timestamp := time.Now().Unix()
	teamName := fmt.Sprintf("e2e_user_team_%d", timestamp)
	userID := fmt.Sprintf("test_user_%d", timestamp)

	t.Run("SetupTeam", func(t *testing.T) {
		payload := map[string]interface{}{
			"team_name": teamName,
			"members": []map[string]interface{}{
				{
					"user_id":   userID,
					"username":  "Test User",
					"is_active": true,
				},
			},
		}

		resp := client.Post(t, "/team/add", payload)
		defer func() { _ = resp.Body.Close() }()
		AssertStatusCode(t, resp, http.StatusCreated)
	})

	t.Run("DeactivateUser", func(t *testing.T) {
		payload := map[string]interface{}{
			"user_id":   userID,
			"is_active": false,
		}

		resp := client.Post(t, "/users/setIsActive", payload)
		defer func() { _ = resp.Body.Close() }()

		AssertStatusCode(t, resp, http.StatusOK)

		var result map[string]interface{}
		client.DecodeJSON(t, resp, &result)

		assert.Contains(t, result, "user")
		user := result["user"].(map[string]interface{})
		assert.Equal(t, userID, user["user_id"])
		assert.Equal(t, false, user["is_active"])
	})

	t.Run("ActivateUser", func(t *testing.T) {
		payload := map[string]interface{}{
			"user_id":   userID,
			"is_active": true,
		}

		resp := client.Post(t, "/users/setIsActive", payload)
		defer func() { _ = resp.Body.Close() }()

		AssertStatusCode(t, resp, http.StatusOK)

		var result map[string]interface{}
		client.DecodeJSON(t, resp, &result)

		assert.Contains(t, result, "user")
		user := result["user"].(map[string]interface{})
		assert.Equal(t, userID, user["user_id"])
		assert.Equal(t, true, user["is_active"])
	})
}

// TestUserReviews тестирует получение PR для ревьюера
func TestUserReviews(t *testing.T) {
	client := NewTestClient()

	timestamp := time.Now().Unix()
	teamName := fmt.Sprintf("e2e_review_team_%d", timestamp)
	authorID := fmt.Sprintf("author_rev_%d", timestamp)
	reviewerID := fmt.Sprintf("reviewer_%d", timestamp)
	prID := fmt.Sprintf("pr_rev_%d", timestamp)

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
					"user_id":   reviewerID,
					"username":  "Reviewer",
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
			"pull_request_name": fmt.Sprintf("PR for review %d", timestamp),
			"author_id":         authorID,
		}

		resp := client.Post(t, "/pullRequest/create", payload)
		defer func() { _ = resp.Body.Close() }()
		AssertStatusCode(t, resp, http.StatusCreated)
	})

	t.Run("GetReviewsForNonExistentUser", func(t *testing.T) {
		resp := client.Get(t, "/users/getReview?user_id=nonexistent_user_999999")
		defer func() { _ = resp.Body.Close() }()

		// Может вернуть 200 с пустым списком или 404
		if resp.StatusCode == http.StatusOK {
			var result map[string]interface{}
			client.DecodeJSON(t, resp, &result)
			assert.Contains(t, result, "user_id")
		}
	})

	t.Run("GetReviewsForReviewer", func(t *testing.T) {
		resp := client.Get(t, fmt.Sprintf("/users/getReview?user_id=%s", reviewerID))
		defer func() { _ = resp.Body.Close() }()

		AssertStatusCode(t, resp, http.StatusOK)

		var result map[string]interface{}
		client.DecodeJSON(t, resp, &result)

		assert.Contains(t, result, "user_id")
		// Может быть назначен или не назначен в зависимости от алгоритма
	})
}

// TestInactiveUserNotAssigned тестирует что неактивный пользователь не получает PR на ревью
func TestInactiveUserNotAssigned(t *testing.T) {
	client := NewTestClient()

	timestamp := time.Now().Unix()
	teamName := fmt.Sprintf("e2e_inactive_team_%d", timestamp)
	authorID := fmt.Sprintf("inactive_author_%d", timestamp)
	inactiveReviewerID := fmt.Sprintf("inactive_rev_%d", timestamp)
	inactivePRID := fmt.Sprintf("inactive_pr_%d", timestamp)

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
					"user_id":   inactiveReviewerID,
					"username":  "Inactive Reviewer",
					"is_active": true,
				},
			},
		}

		resp := client.Post(t, "/team/add", payload)
		defer func() { _ = resp.Body.Close() }()
		AssertStatusCode(t, resp, http.StatusCreated)
	})

	t.Run("DeactivateReviewer", func(t *testing.T) {
		payload := map[string]interface{}{
			"user_id":   inactiveReviewerID,
			"is_active": false,
		}

		resp := client.Post(t, "/users/setIsActive", payload)
		defer func() { _ = resp.Body.Close() }()
		AssertStatusCode(t, resp, http.StatusOK)
	})

	t.Run("CreatePR", func(t *testing.T) {
		payload := map[string]interface{}{
			"pull_request_id":   inactivePRID,
			"pull_request_name": fmt.Sprintf("PR without inactive reviewer %d", timestamp),
			"author_id":         authorID,
		}

		resp := client.Post(t, "/pullRequest/create", payload)
		defer func() { _ = resp.Body.Close() }()
		AssertStatusCode(t, resp, http.StatusCreated)

		var result map[string]interface{}
		client.DecodeJSON(t, resp, &result)

		pr := result["pr"].(map[string]interface{})
		// Не должно быть ревьюеров, так как единственный возможный ревьюер неактивен
		if reviewers, ok := pr["assigned_reviewers"]; ok && reviewers != nil {
			reviewerList := reviewers.([]interface{})
			assert.Empty(t, reviewerList, "Inactive user should not be assigned as reviewer")
		}
	})
}
