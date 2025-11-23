// Имплементация репозитория для работы с pull request'ами в базе данных postgresql
package repository

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"test_avito/internal/database/db"
	"test_avito/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PullRequestRepositoryImpl struct {
	queries *db.Queries
	pool    *pgxpool.Pool
	logger  *slog.Logger
}

func NewPullRequestRepository(pool *pgxpool.Pool, logger *slog.Logger) *PullRequestRepositoryImpl {
	return &PullRequestRepositoryImpl{
		queries: db.New(pool),
		pool:    pool,
		logger:  logger,
	}
}

// Create creates a new pull request with reviewers
func (r *PullRequestRepositoryImpl) Create(ctx context.Context, pr *domain.PullRequest) error {
	txCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	tx, err := r.pool.Begin(txCtx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(context.Background())
			r.logger.Error("panic in Create transaction",
				slog.String("pr_id", pr.ID),
				slog.Any("panic", p),
			)
			panic(p)
		}
		_ = tx.Rollback(context.Background())
	}()

	qtx := r.queries.WithTx(tx)

	createdAt := pgtype.Timestamptz{Valid: false}
	if pr.CreatedAt != nil {
		createdAt = pgtype.Timestamptz{Time: *pr.CreatedAt, Valid: true}
	}
	mergedAt := pgtype.Timestamptz{Valid: false}
	if pr.MergedAt != nil {
		mergedAt = pgtype.Timestamptz{Time: *pr.MergedAt, Valid: true}
	}

	err = qtx.CreatePullRequest(txCtx, db.CreatePullRequestParams{
		ID:        pr.ID,
		Name:      pr.Name,
		AuthorID:  pr.AuthorID,
		Status:    string(pr.Status),
		CreatedAt: createdAt,
		MergedAt:  mergedAt,
	})
	if err != nil {
		r.logger.Error("failed to create PR",
			slog.String("pr_id", pr.ID),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to create PR: %w", err)
	}

	reviewers := make([]string, len(pr.AssignedReviewers))
	copy(reviewers, pr.AssignedReviewers)
	sort.Strings(reviewers)

	for _, reviewerID := range reviewers {
		err = qtx.AddReviewer(txCtx, db.AddReviewerParams{
			PullRequestID: pr.ID,
			ReviewerID:    reviewerID,
			AssignedAt:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
		})
		if err != nil {
			r.logger.Error("failed to add reviewer",
				slog.String("pr_id", pr.ID),
				slog.String("reviewer_id", reviewerID),
				slog.String("error", err.Error()),
			)
			return fmt.Errorf("failed to add reviewer: %w", err)
		}
	}

	if err := tx.Commit(txCtx); err != nil {
		r.logger.Error("failed to commit transaction",
			slog.String("pr_id", pr.ID),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	r.logger.Info("PR created in transaction",
		slog.String("pr_id", pr.ID),
		slog.Int("reviewers_count", len(pr.AssignedReviewers)),
	)
	return nil
}

// GetByID retrieves a pull request by ID with reviewers
func (r *PullRequestRepositoryImpl) GetByID(ctx context.Context, id string) (*domain.PullRequest, error) {
	dbPR, err := r.queries.GetPullRequestByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrPRNotFound
		}
		r.logger.Error("failed to get PR",
			slog.String("pr_id", id),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("failed to get PR: %w", err)
	}

	reviewerIDs, err := r.queries.GetReviewersByPRID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get reviewers: %w", err)
	}

	pr := &domain.PullRequest{
		ID:                dbPR.ID,
		Name:              dbPR.Name,
		AuthorID:          dbPR.AuthorID,
		Status:            domain.PRStatus(dbPR.Status),
		AssignedReviewers: reviewerIDs,
	}

	if dbPR.CreatedAt.Valid {
		pr.CreatedAt = &dbPR.CreatedAt.Time
	}
	if dbPR.MergedAt.Valid {
		pr.MergedAt = &dbPR.MergedAt.Time
	}

	return pr, nil
}

// Update updates an existing pull request
func (r *PullRequestRepositoryImpl) Update(ctx context.Context, pr *domain.PullRequest) error {
	mergedAt := pgtype.Timestamptz{Valid: false}
	if pr.MergedAt != nil {
		mergedAt = pgtype.Timestamptz{Time: *pr.MergedAt, Valid: true}
	}

	err := r.queries.UpdatePullRequest(ctx, db.UpdatePullRequestParams{
		ID:       pr.ID,
		Name:     pr.Name,
		AuthorID: pr.AuthorID,
		Status:   string(pr.Status),
		MergedAt: mergedAt,
	})
	if err != nil {
		r.logger.Error("failed to update PR",
			slog.String("pr_id", pr.ID),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to update PR: %w", err)
	}

	r.logger.Info("PR updated", slog.String("pr_id", pr.ID))
	return nil
}

// Merge marks a PR as merged (idempotent - just one UPDATE)
func (r *PullRequestRepositoryImpl) Merge(ctx context.Context, id string) (*domain.PullRequest, error) {
	now := time.Now()
	mergedPR, err := r.queries.MergePullRequest(ctx, db.MergePullRequestParams{
		ID:       id,
		MergedAt: pgtype.Timestamptz{Time: now, Valid: true},
	})

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			dbPR, getErr := r.queries.GetPullRequestByID(ctx, id)
			if getErr != nil {
				if errors.Is(getErr, pgx.ErrNoRows) {
					return nil, domain.ErrPRNotFound
				}
				return nil, fmt.Errorf("failed to get PR: %w", getErr)
			}

			reviewerIDs, err := r.queries.GetReviewersByPRID(ctx, id)
			if err != nil {
				return nil, fmt.Errorf("failed to get reviewers: %w", err)
			}

			pr := &domain.PullRequest{
				ID:                dbPR.ID,
				Name:              dbPR.Name,
				AuthorID:          dbPR.AuthorID,
				Status:            domain.PRStatus(dbPR.Status),
				AssignedReviewers: reviewerIDs,
			}

			if dbPR.CreatedAt.Valid {
				pr.CreatedAt = &dbPR.CreatedAt.Time
			}
			if dbPR.MergedAt.Valid {
				pr.MergedAt = &dbPR.MergedAt.Time
			}

			r.logger.Info("PR already merged (idempotent)", slog.String("pr_id", id))
			return pr, nil
		}

		r.logger.Error("failed to merge PR",
			slog.String("pr_id", id),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("failed to merge PR: %w", err)
	}

	reviewerIDs, err := r.queries.GetReviewersByPRID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get reviewers: %w", err)
	}

	pr := &domain.PullRequest{
		ID:                mergedPR.ID,
		Name:              mergedPR.Name,
		AuthorID:          mergedPR.AuthorID,
		Status:            domain.PRStatus(mergedPR.Status),
		AssignedReviewers: reviewerIDs,
	}

	if mergedPR.CreatedAt.Valid {
		pr.CreatedAt = &mergedPR.CreatedAt.Time
	}
	if mergedPR.MergedAt.Valid {
		pr.MergedAt = &mergedPR.MergedAt.Time
	}

	r.logger.Info("PR merged", slog.String("pr_id", id))
	return pr, nil
}

// AddReviewer adds a reviewer to a PR
func (r *PullRequestRepositoryImpl) AddReviewer(ctx context.Context, prID, reviewerID string) error {
	err := r.queries.AddReviewer(ctx, db.AddReviewerParams{
		PullRequestID: prID,
		ReviewerID:    reviewerID,
		AssignedAt:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
	})
	if err != nil {
		r.logger.Error("failed to add reviewer",
			slog.String("pr_id", prID),
			slog.String("reviewer_id", reviewerID),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to add reviewer: %w", err)
	}

	r.logger.Info("reviewer added",
		slog.String("pr_id", prID),
		slog.String("reviewer_id", reviewerID),
	)
	return nil
}

// RemoveReviewer removes a reviewer from a PR
func (r *PullRequestRepositoryImpl) RemoveReviewer(ctx context.Context, prID, reviewerID string) error {
	err := r.queries.RemoveReviewer(ctx, db.RemoveReviewerParams{
		PullRequestID: prID,
		ReviewerID:    reviewerID,
	})
	if err != nil {
		r.logger.Error("failed to remove reviewer",
			slog.String("pr_id", prID),
			slog.String("reviewer_id", reviewerID),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to remove reviewer: %w", err)
	}

	r.logger.Info("reviewer removed",
		slog.String("pr_id", prID),
		slog.String("reviewer_id", reviewerID),
	)
	return nil
}

// ReassignReviewer replaces old reviewer with new one in a transaction
func (r *PullRequestRepositoryImpl) ReassignReviewer(ctx context.Context, prID, oldReviewerID, newReviewerID string) error {
	txCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	tx, err := r.pool.Begin(txCtx)
	if err != nil {
		r.logger.Error("failed to begin transaction",
			slog.String("pr_id", prID),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(context.Background())
			r.logger.Error("panic in ReassignReviewer transaction",
				slog.String("pr_id", prID),
				slog.Any("panic", p),
			)
			panic(p)
		}
		_ = tx.Rollback(context.Background())
	}()

	qtx := r.queries.WithTx(tx)

	err = qtx.RemoveReviewer(txCtx, db.RemoveReviewerParams{
		PullRequestID: prID,
		ReviewerID:    oldReviewerID,
	})
	if err != nil {
		r.logger.Error("failed to remove reviewer in transaction",
			slog.String("pr_id", prID),
			slog.String("reviewer_id", oldReviewerID),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to remove reviewer: %w", err)
	}

	err = qtx.AddReviewer(txCtx, db.AddReviewerParams{
		PullRequestID: prID,
		ReviewerID:    newReviewerID,
		AssignedAt:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
	})
	if err != nil {
		r.logger.Error("failed to add reviewer in transaction",
			slog.String("pr_id", prID),
			slog.String("reviewer_id", newReviewerID),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to add reviewer: %w", err)
	}

	if err := tx.Commit(txCtx); err != nil {
		r.logger.Error("failed to commit transaction",
			slog.String("pr_id", prID),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	r.logger.Info("reviewer reassigned in transaction",
		slog.String("pr_id", prID),
		slog.String("old_reviewer_id", oldReviewerID),
		slog.String("new_reviewer_id", newReviewerID),
	)
	return nil
}

// AssignReviewers assigns reviewers to an existing PR in a transaction
// Returns error if PR already has any reviewers assigned
func (r *PullRequestRepositoryImpl) AssignReviewers(ctx context.Context, prID string, reviewerIDs []string) error {
	// Add timeout for transaction
	txCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	tx, err := r.pool.Begin(txCtx)
	if err != nil {
		r.logger.Error("failed to begin transaction",
			slog.String("pr_id", prID),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(context.Background())
			r.logger.Error("panic in AssignReviewers transaction",
				slog.String("pr_id", prID),
				slog.Any("panic", p),
			)
			panic(p)
		}
		_ = tx.Rollback(context.Background())
	}()

	qtx := r.queries.WithTx(tx)

	existingReviewers, err := qtx.GetReviewersByPRID(txCtx, prID)
	if err != nil {
		r.logger.Error("failed to check existing reviewers",
			slog.String("pr_id", prID),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to check existing reviewers: %w", err)
	}

	if len(existingReviewers) > 0 {
		r.logger.Error("PR already has reviewers assigned",
			slog.String("pr_id", prID),
			slog.Int("existing_count", len(existingReviewers)),
		)
		return domain.ErrReviewersAlreadyAssigned
	}

	sortedReviewers := make([]string, len(reviewerIDs))
	copy(sortedReviewers, reviewerIDs)
	sort.Strings(sortedReviewers)

	now := pgtype.Timestamptz{Time: time.Now(), Valid: true}
	for _, reviewerID := range sortedReviewers {
		err = qtx.AddReviewer(txCtx, db.AddReviewerParams{
			PullRequestID: prID,
			ReviewerID:    reviewerID,
			AssignedAt:    now,
		})
		if err != nil {
			r.logger.Error("failed to add reviewer in transaction",
				slog.String("pr_id", prID),
				slog.String("reviewer_id", reviewerID),
				slog.String("error", err.Error()),
			)
			return fmt.Errorf("failed to add reviewer: %w", err)
		}
	}

	if err := tx.Commit(txCtx); err != nil {
		r.logger.Error("failed to commit transaction",
			slog.String("pr_id", prID),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	r.logger.Info("reviewers assigned in transaction",
		slog.String("pr_id", prID),
		slog.Int("count", len(reviewerIDs)),
	)
	return nil
}

// GetReviewersByPRID gets all reviewers for a PR
func (r *PullRequestRepositoryImpl) GetReviewersByPRID(ctx context.Context, prID string) ([]string, error) {
	reviewers, err := r.queries.GetReviewersByPRID(ctx, prID)
	if err != nil {
		r.logger.Error("failed to get reviewers",
			slog.String("pr_id", prID),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("failed to get reviewers: %w", err)
	}

	return reviewers, nil
}

// GetPRsByReviewer gets all PRs assigned to a reviewer
func (r *PullRequestRepositoryImpl) GetPRsByReviewer(ctx context.Context, reviewerID string) ([]domain.PullRequestShort, error) {
	dbPRs, err := r.queries.GetPRsByReviewer(ctx, reviewerID)
	if err != nil {
		r.logger.Error("failed to get PRs by reviewer",
			slog.String("reviewer_id", reviewerID),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("failed to get PRs by reviewer: %w", err)
	}

	prs := make([]domain.PullRequestShort, len(dbPRs))
	for i, dbPR := range dbPRs {
		prs[i] = domain.PullRequestShort{
			ID:       dbPR.ID,
			Name:     dbPR.Name,
			AuthorID: dbPR.AuthorID,
			Status:   domain.PRStatus(dbPR.Status),
		}
	}

	return prs, nil
}

// Exists checks if a PR exists
func (r *PullRequestRepositoryImpl) Exists(ctx context.Context, id string) (bool, error) {
	exists, err := r.queries.PullRequestExists(ctx, id)
	if err != nil {
		r.logger.Error("failed to check PR existence",
			slog.String("pr_id", id),
			slog.String("error", err.Error()),
		)
		return false, fmt.Errorf("failed to check PR existence: %w", err)
	}

	return exists, nil
}

// Count returns total number of PRs
func (r *PullRequestRepositoryImpl) Count(ctx context.Context) (int, error) {
	count, err := r.queries.CountPullRequests(ctx)
	if err != nil {
		r.logger.Error("failed to count PRs", slog.String("error", err.Error()))
		return 0, fmt.Errorf("failed to count PRs: %w", err)
	}

	return int(count), nil
}

// CountByStatus returns number of PRs by status
func (r *PullRequestRepositoryImpl) CountByStatus(ctx context.Context, status domain.PRStatus) (int, error) {
	count, err := r.queries.CountPullRequestsByStatus(ctx, string(status))
	if err != nil {
		r.logger.Error("failed to count PRs by status",
			slog.String("status", string(status)),
			slog.String("error", err.Error()),
		)
		return 0, fmt.Errorf("failed to count PRs by status: %w", err)
	}

	return int(count), nil
}
