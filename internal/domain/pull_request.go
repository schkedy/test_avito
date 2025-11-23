package domain

import "time"

type PRStatus string

const (
	PRStatusOpen   PRStatus = "OPEN"
	PRStatusMerged PRStatus = "MERGED"
)

type PullRequest struct {
	ID                string     `json:"pull_request_id"`
	Name              string     `json:"pull_request_name"`
	AuthorID          string     `json:"author_id"`
	Status            PRStatus   `json:"status"`
	AssignedReviewers []string   `json:"assigned_reviewers"`
	CreatedAt         *time.Time `json:"createdAt,omitempty"`
	MergedAt          *time.Time `json:"mergedAt,omitempty"`
}

type PullRequestShort struct {
	ID       string   `json:"pull_request_id"`
	Name     string   `json:"pull_request_name"`
	AuthorID string   `json:"author_id"`
	Status   PRStatus `json:"status"`
}

func NewPullRequest(id, name, authorID string) *PullRequest {
	now := time.Now()
	return &PullRequest{
		ID:                id,
		Name:              name,
		AuthorID:          authorID,
		Status:            PRStatusOpen,
		AssignedReviewers: make([]string, 0, 2),
		CreatedAt:         &now,
	}
}

func (pr *PullRequest) Validate() error {
	if pr.ID == "" {
		return ErrInvalidInput
	}
	if pr.Name == "" {
		return ErrInvalidInput
	}
	if pr.AuthorID == "" {
		return ErrInvalidInput
	}
	if pr.Status != PRStatusOpen && pr.Status != PRStatusMerged {
		return ErrInvalidPRStatus
	}
	return nil
}

func (pr *PullRequest) IsMerged() bool {
	return pr.Status == PRStatusMerged
}

func (pr *PullRequest) Merge() error {
	if pr.IsMerged() {
		return nil // Idempotent
	}
	now := time.Now()
	pr.Status = PRStatusMerged
	pr.MergedAt = &now
	return nil
}

func (pr *PullRequest) HasReviewer(userID string) bool {
	for _, reviewerID := range pr.AssignedReviewers {
		if reviewerID == userID {
			return true
		}
	}
	return false
}

func (pr *PullRequest) AddReviewer(userID string) error {
	if pr.IsMerged() {
		return ErrPRMerged
	}
	if pr.HasReviewer(userID) {
		return nil // Already assigned
	}
	if len(pr.AssignedReviewers) >= 2 {
		return ErrInvalidInput
	}
	pr.AssignedReviewers = append(pr.AssignedReviewers, userID)
	return nil
}

func (pr *PullRequest) RemoveReviewer(userID string) error {
	if pr.IsMerged() {
		return ErrPRMerged
	}
	if !pr.HasReviewer(userID) {
		return ErrReviewerNotFound
	}

	newReviewers := make([]string, 0, len(pr.AssignedReviewers)-1)
	for _, reviewerID := range pr.AssignedReviewers {
		if reviewerID != userID {
			newReviewers = append(newReviewers, reviewerID)
		}
	}
	pr.AssignedReviewers = newReviewers
	return nil
}

func (pr *PullRequest) ToShort() *PullRequestShort {
	return &PullRequestShort{
		ID:       pr.ID,
		Name:     pr.Name,
		AuthorID: pr.AuthorID,
		Status:   pr.Status,
	}
}
