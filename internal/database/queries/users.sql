-- name: CreateUser :exec
INSERT INTO users (id, username, team_name, is_active)
VALUES ($1, $2, $3, $4);

-- name: UpdateUser :exec
UPDATE users 
SET username = $2, team_name = $3, is_active = $4
WHERE id = $1;

-- name: UpsertUser :exec
INSERT INTO users (id, username, team_name, is_active)
VALUES ($1, $2, $3, $4)
ON CONFLICT (id) 
DO UPDATE SET 
    username = EXCLUDED.username,
    team_name = EXCLUDED.team_name,
    is_active = EXCLUDED.is_active;

-- name: GetUserByID :one
SELECT id, username, team_name, is_active 
FROM users 
WHERE id = $1;

-- name: GetUsersByTeam :many
SELECT id, username, team_name, is_active 
FROM users 
WHERE team_name = $1
ORDER BY username;

-- name: SetUserIsActive :exec
UPDATE users SET is_active = $2 WHERE id = $1;

-- name: DeactivateTeamUsers :execrows
UPDATE users SET is_active = false WHERE team_name = $1 AND is_active = true;

-- name: GetActiveUsersByTeam :many
SELECT id, username, team_name, is_active 
FROM users 
WHERE team_name = $1 AND is_active = true AND id != $2
ORDER BY username;

-- name: UserExists :one
SELECT EXISTS(SELECT 1 FROM users WHERE id = $1);

-- name: CountUsers :one
SELECT COUNT(*) FROM users;

-- name: CountActiveUsers :one
SELECT COUNT(*) FROM users WHERE is_active = true;
