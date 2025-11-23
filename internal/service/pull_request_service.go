package service

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"

	"test_avito/internal/domain"
	"test_avito/internal/repository"
)

type PullRequestService struct {
	prRepo   repository.PullRequestRepository
	userRepo repository.UserRepository
	logger   *slog.Logger
}

func NewPullRequestService(
	prRepo repository.PullRequestRepository,
	userRepo repository.UserRepository,
	logger *slog.Logger,
) *PullRequestService {
	return &PullRequestService{
		prRepo:   prRepo,
		userRepo: userRepo,
		logger:   logger,
	}
}

// CreatePR creates a new PR and automatically assigns up to 2 reviewers from author's team
func (s *PullRequestService) CreatePR(ctx context.Context, prID, prName, authorID string) (*domain.PullRequest, error) {
	if prID == "" || prName == "" || authorID == "" {
		return nil, domain.ErrInvalidInput
	}

	s.logger.Info("creating PR",
		slog.String("pr_id", prID),
		slog.String("author_id", authorID),
	)

	exists, err := s.prRepo.Exists(ctx, prID)
	if err != nil {
		return nil, fmt.Errorf("failed to check PR existence: %w", err)
	}
	if exists {
		return nil, domain.ErrPRExists
	}

	author, err := s.userRepo.GetByID(ctx, authorID)
	if err != nil {
		return nil, fmt.Errorf("author not found: %w", err)
	}

	activeMembers, err := s.userRepo.GetActiveByTeam(ctx, author.TeamName, authorID)
	if err != nil {
		return nil, fmt.Errorf("failed to get team members: %w", err)
	}

	pr := domain.NewPullRequest(prID, prName, authorID)

	reviewers := s.selectRandomReviewers(activeMembers, 2)
	pr.AssignedReviewers = reviewers

	if err := s.prRepo.Create(ctx, pr); err != nil {
		return nil, fmt.Errorf("failed to create PR: %w", err)
	}

	s.logger.Info("PR created",
		slog.String("pr_id", prID),
		slog.Int("reviewers_assigned", len(reviewers)),
	)

	return pr, nil
}

func (s *PullRequestService) MergePR(ctx context.Context, prID string) (*domain.PullRequest, error) {
	if prID == "" {
		return nil, domain.ErrInvalidInput
	}

	s.logger.Info("merging PR", slog.String("pr_id", prID))

	pr, err := s.prRepo.Merge(ctx, prID)
	if err != nil {
		return nil, fmt.Errorf("failed to merge PR: %w", err)
	}

	s.logger.Info("PR merged", slog.String("pr_id", prID))

	return pr, nil
}

func (s *PullRequestService) ReassignReviewer(ctx context.Context, prID, oldReviewerID string) (string, *domain.PullRequest, error) {
	if prID == "" || oldReviewerID == "" {
		return "", nil, domain.ErrInvalidInput
	}

	s.logger.Info("reassigning reviewer",
		slog.String("pr_id", prID),
		slog.String("old_reviewer_id", oldReviewerID),
	)

	pr, err := s.prRepo.GetByID(ctx, prID)
	if err != nil {
		return "", nil, err
	}

	if pr.IsMerged() {
		return "", nil, domain.ErrPRMerged
	}

	if !pr.HasReviewer(oldReviewerID) {
		return "", nil, domain.ErrReviewerNotFound
	}

	oldReviewer, err := s.userRepo.GetByID(ctx, oldReviewerID)
	if err != nil {
		return "", nil, fmt.Errorf("old reviewer not found: %w", err)
	}

	// Get active members from the reviewer's team, excluding:
	// - the old reviewer
	// - the PR author
	// - current reviewers
	activeMembers, err := s.userRepo.GetActiveByTeam(ctx, oldReviewer.TeamName, "")
	if err != nil {
		return "", nil, fmt.Errorf("failed to get team members: %w", err)
	}

	candidates := make([]domain.User, 0)
	for _, member := range activeMembers {
		if member.ID == pr.AuthorID {
			continue
		}
		if member.ID == oldReviewerID {
			continue
		}
		isCurrentReviewer := false
		for _, reviewerID := range pr.AssignedReviewers {
			if member.ID == reviewerID {
				isCurrentReviewer = true
				break
			}
		}
		if !isCurrentReviewer {
			candidates = append(candidates, member)
		}
	}

	if len(candidates) == 0 {
		return "", nil, domain.ErrNoAvailableReviewer
	}

	newReviewerID := s.selectRandomReviewers(candidates, 1)[0]

	// Reassign reviewer in transaction (remove old + add new atomically)
	if err := s.prRepo.ReassignReviewer(ctx, prID, oldReviewerID, newReviewerID); err != nil {
		return "", nil, fmt.Errorf("failed to reassign reviewer: %w", err)
	}

	pr, err = s.prRepo.GetByID(ctx, prID)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get updated PR: %w", err)
	}

	s.logger.Info("reviewer reassigned",
		slog.String("pr_id", prID),
		slog.String("old_reviewer_id", oldReviewerID),
		slog.String("new_reviewer_id", newReviewerID),
	)

	return newReviewerID, pr, nil
}

// GetPRsByReviewer retrieves all PRs assigned to a specific reviewer
func (s *PullRequestService) GetPRsByReviewer(ctx context.Context, reviewerID string) ([]domain.PullRequestShort, error) {
	if reviewerID == "" {
		return nil, domain.ErrInvalidInput
	}

	_, err := s.userRepo.GetByID(ctx, reviewerID)
	if err != nil {
		return nil, err
	}

	s.logger.Info("getting PRs for reviewer", slog.String("reviewer_id", reviewerID))

	prs, err := s.prRepo.GetPRsByReviewer(ctx, reviewerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get PRs by reviewer: %w", err)
	}

	return prs, nil
}

// selectRandomReviewers selects up to maxCount random reviewers from candidates
func (s *PullRequestService) selectRandomReviewers(candidates []domain.User, maxCount int) []string {
	if len(candidates) == 0 {
		return []string{}
	}

	count := maxCount
	if len(candidates) < count {
		count = len(candidates)
	}

	shuffled := make([]domain.User, len(candidates))
	copy(shuffled, candidates)

	// Fisher-Yates shuffle
	// #nosec G404 -- math/rand is sufficient for reviewer selection (not security-sensitive)
	for i := len(shuffled) - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	}

	reviewers := make([]string, count)
	for i := 0; i < count; i++ {
		reviewers[i] = shuffled[i].ID
	}

	return reviewers
}

// AssignReviewersToPR assigns reviewers to an existing PR (must have no reviewers yet)
// If reviewerIDs is empty or nil, assigns random reviewers from author's team (up to 2)
// If reviewerIDs is provided, assigns those specific reviewers
func (s *PullRequestService) AssignReviewersToPR(ctx context.Context, prID string, reviewerIDs []string) (*domain.PullRequest, error) {
	if prID == "" {
		return nil, domain.ErrInvalidInput
	}

	s.logger.Info("assigning reviewers to PR",
		slog.String("pr_id", prID),
		slog.Int("specified_count", len(reviewerIDs)),
	)

	pr, err := s.prRepo.GetByID(ctx, prID)
	if err != nil {
		return nil, err
	}

	if pr.IsMerged() {
		return nil, domain.ErrPRMerged
	}

	var reviewersToAssign []string

	if len(reviewerIDs) == 0 {
		author, err := s.userRepo.GetByID(ctx, pr.AuthorID)
		if err != nil {
			return nil, err
		}

		activeMembers, err := s.userRepo.GetActiveByTeam(ctx, author.TeamName, pr.AuthorID)
		if err != nil {
			return nil, err
		}

		if len(activeMembers) == 0 {
			return nil, domain.ErrNoAvailableReviewer
		}

		reviewersToAssign = s.selectRandomReviewers(activeMembers, 2)
	} else {
		if len(reviewerIDs) > 2 {
			return nil, domain.ErrInvalidInput
		}

		author, err := s.userRepo.GetByID(ctx, pr.AuthorID)
		if err != nil {
			return nil, err
		}

		for _, reviewerID := range reviewerIDs {
			user, err := s.userRepo.GetByID(ctx, reviewerID)
			if err != nil {
				return nil, err
			}
			if !user.IsActive {
				s.logger.Warn("reviewer is not active",
					slog.String("reviewer_id", reviewerID),
				)
				return nil, domain.ErrUserNotActive
			}
			if user.TeamName != author.TeamName {
				s.logger.Warn("reviewer not in author's team",
					slog.String("reviewer_id", reviewerID),
					slog.String("reviewer_team", user.TeamName),
					slog.String("author_team", author.TeamName),
				)
				return nil, domain.ErrReviewerNotInTeam
			}
			if reviewerID == pr.AuthorID {
				s.logger.Warn("attempt to assign author as reviewer",
					slog.String("author_id", pr.AuthorID),
				)
				return nil, domain.ErrAuthorAsReviewer
			}
		}

		reviewersToAssign = reviewerIDs
	}

	// Assign reviewers (repository will check if PR already has reviewers)
	if err := s.prRepo.AssignReviewers(ctx, prID, reviewersToAssign); err != nil {
		return nil, fmt.Errorf("failed to assign reviewers: %w", err)
	}

	pr, err = s.prRepo.GetByID(ctx, prID)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated PR: %w", err)
	}

	s.logger.Info("reviewers assigned to PR",
		slog.String("pr_id", prID),
		slog.Int("assigned_count", len(reviewersToAssign)),
		slog.Int("total_reviewers", len(pr.AssignedReviewers)),
	)

	return pr, nil
}
