-- name: CreateTeam :exec
INSERT INTO teams (name) VALUES ($1);

-- name: GetTeamByName :one
SELECT name FROM teams WHERE name = $1;

-- name: TeamExists :one
SELECT EXISTS(SELECT 1 FROM teams WHERE name = $1);

-- name: CountTeams :one
SELECT COUNT(*) FROM teams;
