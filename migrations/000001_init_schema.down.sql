-- Drop indexes
DROP INDEX IF EXISTS idx_pr_reviewers_reviewer;
DROP INDEX IF EXISTS idx_pr_status;
DROP INDEX IF EXISTS idx_pr_author;
DROP INDEX IF EXISTS idx_users_team;

-- Drop tables (in reverse order due to foreign keys)
DROP TABLE IF EXISTS pr_reviewers;
DROP TABLE IF EXISTS pull_requests;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS teams;
