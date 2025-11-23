-- name: CreatePullRequest :exec
INSERT INTO pull_requests (id, name, author_id, status, created_at, merged_at)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: GetPullRequestByID :one
SELECT id, name, author_id, status, created_at, merged_at
FROM pull_requests
WHERE id = $1;

-- name: UpdatePullRequest :exec
UPDATE pull_requests
SET name = $2, author_id = $3, status = $4, merged_at = $5
WHERE id = $1;

-- name: MergePullRequest :one
UPDATE pull_requests
SET status = 'MERGED', merged_at = $2
WHERE id = $1 AND status != 'MERGED'
RETURNING id, name, author_id, status, created_at, merged_at;

-- name: PullRequestExists :one
SELECT EXISTS(SELECT 1 FROM pull_requests WHERE id = $1);

-- name: CountPullRequests :one
SELECT COUNT(*) FROM pull_requests;

-- name: CountPullRequestsByStatus :one
SELECT COUNT(*) FROM pull_requests WHERE status = $1;

-- name: AddReviewer :exec
INSERT INTO pr_reviewers (pull_request_id, reviewer_id, assigned_at)
VALUES ($1, $2, $3)
ON CONFLICT (pull_request_id, reviewer_id) DO NOTHING;

-- name: RemoveReviewer :exec
DELETE FROM pr_reviewers
WHERE pull_request_id = $1 AND reviewer_id = $2;

-- name: GetReviewersByPRID :many
SELECT reviewer_id
FROM pr_reviewers
WHERE pull_request_id = $1
ORDER BY assigned_at;

-- name: GetPRsByReviewer :many
SELECT DISTINCT pr.id, pr.name, pr.author_id, pr.status, pr.created_at
FROM pull_requests pr
INNER JOIN pr_reviewers prr ON pr.id = prr.pull_request_id
WHERE prr.reviewer_id = $1
ORDER BY pr.created_at DESC;
