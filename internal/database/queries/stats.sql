-- name: GetStats :one
SELECT 
    (SELECT COUNT(*) FROM pull_requests) as total_prs,
    (SELECT COUNT(*) FROM pull_requests WHERE status = 'OPEN') as open_prs,
    (SELECT COUNT(*) FROM pull_requests WHERE status = 'MERGED') as merged_prs,
    (SELECT COUNT(*) FROM teams) as total_teams,
    (SELECT COUNT(*) FROM users) as total_users,
    (SELECT COUNT(*) FROM users WHERE is_active = true) as active_users;
