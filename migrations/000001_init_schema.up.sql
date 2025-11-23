-- Teams table
CREATE TABLE IF NOT EXISTS teams (
    name VARCHAR(255) PRIMARY KEY
);

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id VARCHAR(255) PRIMARY KEY,
    username VARCHAR(255) NOT NULL,
    team_name VARCHAR(255) NOT NULL REFERENCES teams(name) ON DELETE CASCADE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE
);

-- Pull requests table
CREATE TABLE IF NOT EXISTS pull_requests (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    author_id VARCHAR(255) NOT NULL REFERENCES users(id),
    status VARCHAR(50) NOT NULL CHECK (status IN ('OPEN', 'MERGED')),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    merged_at TIMESTAMPTZ
);

-- PR reviewers (many-to-many relationship)
CREATE TABLE IF NOT EXISTS pr_reviewers (
    pull_request_id VARCHAR(255) NOT NULL REFERENCES pull_requests(id) ON DELETE CASCADE,
    reviewer_id VARCHAR(255) NOT NULL REFERENCES users(id),
    assigned_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (pull_request_id, reviewer_id)
);

-- Indexes для производительности
CREATE INDEX IF NOT EXISTS idx_users_team ON users(team_name);
CREATE INDEX IF NOT EXISTS idx_pr_author ON pull_requests(author_id);
CREATE INDEX IF NOT EXISTS idx_pr_status ON pull_requests(status);
CREATE INDEX IF NOT EXISTS idx_pr_reviewers_reviewer ON pr_reviewers(reviewer_id);
