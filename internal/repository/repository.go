// Интерфейсы репозиториев для работы с даннымми
package repository

import (
	"context"

	"test_avito/internal/domain"
)

type TeamRepository interface {
	Create(ctx context.Context, team *domain.Team) error
	// CreateWithMembers creates a team with members in a transaction
	CreateWithMembers(ctx context.Context, team *domain.Team) error
	// UpdateMembers updates team members in a transaction
	UpdateMembers(ctx context.Context, teamName string, members []domain.User) error
	// GetByName retrieves a team by name
	GetByName(ctx context.Context, name string) (*domain.Team, error)
	// Exists checks if a team exists
	Exists(ctx context.Context, name string) (bool, error)
	// Count returns the total number of teams
	Count(ctx context.Context) (int, error)
}

type UserRepository interface {
	// Create creates a new user
	Create(ctx context.Context, user *domain.User) error
	// Update updates an existing user
	Update(ctx context.Context, user *domain.User) error
	// GetByID retrieves a user by ID
	GetByID(ctx context.Context, id string) (*domain.User, error)
	// GetByTeam retrieves all users in a team
	GetByTeam(ctx context.Context, teamName string) ([]domain.User, error)
	// SetIsActive updates the user's active status
	SetIsActive(ctx context.Context, userID string, isActive bool) error
	// GetActiveByTeam retrieves all active users in a team excluding specific user
	GetActiveByTeam(ctx context.Context, teamName string, excludeUserID string) ([]domain.User, error)
	// Exists checks if a user exists
	Exists(ctx context.Context, id string) (bool, error)
	// Count returns the total number of users
	Count(ctx context.Context) (int, error)
	// CountActive returns the number of active users
	CountActive(ctx context.Context) (int, error)
	// Upsert creates or updates a user
	Upsert(ctx context.Context, user *domain.User) error
	// DeactivateTeamUsers deactivates all users in a team
	DeactivateTeamUsers(ctx context.Context, teamName string) (int, error)
}

type PullRequestRepository interface {
	// Create creates a new pull request
	Create(ctx context.Context, pr *domain.PullRequest) error
	// GetByID retrieves a pull request by ID
	GetByID(ctx context.Context, id string) (*domain.PullRequest, error)
	// Update updates an existing pull request
	Update(ctx context.Context, pr *domain.PullRequest) error
	// Merge marks a PR as merged (idempotent)
	Merge(ctx context.Context, id string) (*domain.PullRequest, error)
	// AddReviewer adds a reviewer to a PR
	AddReviewer(ctx context.Context, prID, reviewerID string) error
	// RemoveReviewer removes a reviewer from a PR
	RemoveReviewer(ctx context.Context, prID, reviewerID string) error
	// ReassignReviewer replaces old reviewer with new one in a transaction
	ReassignReviewer(ctx context.Context, prID, oldReviewerID, newReviewerID string) error
	// AssignReviewers assigns reviewers to an existing PR in a transaction
	AssignReviewers(ctx context.Context, prID string, reviewerIDs []string) error
	// GetReviewersByPRID gets all reviewers for a PR
	GetReviewersByPRID(ctx context.Context, prID string) ([]string, error)
	// GetPRsByReviewer gets all PRs assigned to a reviewer
	GetPRsByReviewer(ctx context.Context, reviewerID string) ([]domain.PullRequestShort, error)
	// Exists checks if a PR exists
	Exists(ctx context.Context, id string) (bool, error)
	// Count returns total number of PRs
	Count(ctx context.Context) (int, error)
	// CountByStatus returns number of PRs by status
	CountByStatus(ctx context.Context, status domain.PRStatus) (int, error)
}

type StatsRepository interface {
	// GetStats retrieves overall statistics
	GetStats(ctx context.Context) (*Stats, error)
}

// Stats represents overall system statistics
type Stats struct {
	TotalPRs    int `json:"total_prs"`
	OpenPRs     int `json:"open_prs"`
	MergedPRs   int `json:"merged_prs"`
	TotalTeams  int `json:"total_teams"`
	TotalUsers  int `json:"total_users"`
	ActiveUsers int `json:"active_users"`
}
