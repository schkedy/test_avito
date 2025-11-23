package handlers

import (
	"log/slog"
	"net/http"

	"test_avito/internal/domain"
	"test_avito/internal/service"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	teamService  *service.TeamService
	userService  *service.UserService
	prService    *service.PullRequestService
	statsService *service.StatsService
	logger       *slog.Logger
}

func NewHandler(
	teamService *service.TeamService,
	userService *service.UserService,
	prService *service.PullRequestService,
	statsService *service.StatsService,
	logger *slog.Logger,
) *Handler {
	return &Handler{
		teamService:  teamService,
		userService:  userService,
		prService:    prService,
		statsService: statsService,
		logger:       logger,
	}
}

// /health
func (h *Handler) GetHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}

// /stats
func (h *Handler) GetStats(c *gin.Context) {
	stats, err := h.statsService.GetStats(c.Request.Context())
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, stats)
}

// /team/add
func (h *Handler) TeamAdd(c *gin.Context) {
	var req struct {
		TeamName string `json:"team_name" binding:"required"`
		Members  []struct {
			UserID   string `json:"user_id" binding:"required"`
			Username string `json:"username" binding:"required"`
			IsActive bool   `json:"is_active"`
		} `json:"members" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.handleError(c, domain.ErrInvalidInput)
		return
	}

	// Конвертация в доменную модель
	members := make([]domain.User, len(req.Members))
	for i, m := range req.Members {
		members[i] = domain.User{
			ID:       m.UserID,
			Username: m.Username,
			TeamName: req.TeamName,
			IsActive: m.IsActive,
		}
	}

	team := &domain.Team{
		Name:    req.TeamName,
		Members: members,
	}

	// Check if team exists to determine status code
	existingTeam, _ := h.teamService.GetTeam(c.Request.Context(), req.TeamName)

	err := h.teamService.AddTeam(c.Request.Context(), team)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Возвращаем соответствующий статусный код
	status := http.StatusCreated
	if existingTeam != nil {
		status = http.StatusOK
	}

	c.JSON(status, gin.H{
		"team": gin.H{
			"team_name": team.Name,
		},
	})
}

// /team/get
func (h *Handler) TeamGet(c *gin.Context) {
	teamName := c.Query("team_name")
	if teamName == "" {
		h.handleError(c, domain.ErrInvalidInput)
		return
	}

	team, err := h.teamService.GetTeam(c.Request.Context(), teamName)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Конвертация в API-ответ
	members := make([]gin.H, len(team.Members))
	for i, m := range team.Members {
		members[i] = gin.H{
			"user_id":   m.ID,
			"username":  m.Username,
			"is_active": m.IsActive,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"team_name": team.Name,
		"members":   members,
	})
}

// /team/deactivate
func (h *Handler) TeamDeactivate(c *gin.Context) {
	var req struct {
		TeamName string `json:"team_name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.handleError(c, domain.ErrInvalidInput)
		return
	}

	team, deactivatedCount, err := h.teamService.DeactivateTeam(c.Request.Context(), req.TeamName)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Конвертация в API-ответ
	members := make([]gin.H, len(team.Members))
	for i, m := range team.Members {
		members[i] = gin.H{
			"user_id":   m.ID,
			"username":  m.Username,
			"is_active": m.IsActive,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"team": gin.H{
			"team_name": team.Name,
			"members":   members,
		},
		"deactivated_count": deactivatedCount,
	})
}

// /users/setIsActive
func (h *Handler) UsersSetIsActive(c *gin.Context) {
	var req struct {
		UserID   string `json:"user_id" binding:"required"`
		IsActive bool   `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.handleError(c, domain.ErrInvalidInput)
		return
	}

	user, err := h.userService.SetIsActive(c.Request.Context(), req.UserID, req.IsActive)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"user_id":   user.ID,
			"username":  user.Username,
			"team_name": user.TeamName,
			"is_active": user.IsActive,
		},
	})
}

// /pullRequest/create
func (h *Handler) PullRequestCreate(c *gin.Context) {
	var req struct {
		PullRequestID   string `json:"pull_request_id" binding:"required"`
		PullRequestName string `json:"pull_request_name" binding:"required"`
		AuthorID        string `json:"author_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.handleError(c, domain.ErrInvalidInput)
		return
	}

	pr, err := h.prService.CreatePR(c.Request.Context(), req.PullRequestID, req.PullRequestName, req.AuthorID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"pr": h.prToResponse(pr),
	})
}

// /pullRequest/merge
func (h *Handler) PullRequestMerge(c *gin.Context) {
	var req struct {
		PullRequestID string `json:"pull_request_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.handleError(c, domain.ErrInvalidInput)
		return
	}

	pr, err := h.prService.MergePR(c.Request.Context(), req.PullRequestID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"pr": h.prToResponse(pr),
	})
}

// /pullRequest/reassign
func (h *Handler) PullRequestReassign(c *gin.Context) {
	var req struct {
		PullRequestID string `json:"pull_request_id" binding:"required"`
		OldUserID     string `json:"old_user_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.handleError(c, domain.ErrInvalidInput)
		return
	}

	newReviewerID, pr, err := h.prService.ReassignReviewer(c.Request.Context(), req.PullRequestID, req.OldUserID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"pr":          h.prToResponse(pr),
		"replaced_by": newReviewerID,
	})
}

// /pullRequest/assign
func (h *Handler) PullRequestAssign(c *gin.Context) {
	var req struct {
		PullRequestID string `json:"pull_request_id" binding:"required"`
		ReviewerIDs   []struct {
			UserID string `json:"user_id" binding:"required"`
		} `json:"reviewer_ids"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.handleError(c, domain.ErrInvalidInput)
		return
	}

	// Extract user IDs from the request
	var reviewerIDs []string
	if req.ReviewerIDs != nil {
		reviewerIDs = make([]string, len(req.ReviewerIDs))
		for i, r := range req.ReviewerIDs {
			reviewerIDs[i] = r.UserID
		}
	}

	pr, err := h.prService.AssignReviewersToPR(c.Request.Context(), req.PullRequestID, reviewerIDs)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"pr": h.prToResponse(pr),
	})
}

// /users/getReview
func (h *Handler) UsersGetReview(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		h.handleError(c, domain.ErrInvalidInput)
		return
	}

	prs, err := h.prService.GetPRsByReviewer(c.Request.Context(), userID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Convert to API model
	prList := make([]gin.H, len(prs))
	for i, pr := range prs {
		prList[i] = gin.H{
			"pull_request_id":   pr.ID,
			"pull_request_name": pr.Name,
			"author_id":         pr.AuthorID,
			"status":            pr.Status,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id":       userID,
		"pull_requests": prList,
	})
}

// Helper functions

func (h *Handler) prToResponse(pr *domain.PullRequest) gin.H {
	return gin.H{
		"pull_request_id":    pr.ID,
		"pull_request_name":  pr.Name,
		"author_id":          pr.AuthorID,
		"status":             pr.Status,
		"assigned_reviewers": pr.AssignedReviewers,
		"createdAt":          pr.CreatedAt,
		"mergedAt":           pr.MergedAt,
	}
}

func (h *Handler) handleError(c *gin.Context, err error) {
	apiErr := domain.ToAPIError(err)

	var statusCode int
	switch apiErr.Code {
	case domain.CodeBadRequest:
		statusCode = http.StatusBadRequest
	case domain.CodeNotFound:
		statusCode = http.StatusNotFound
	case domain.CodePRExists, domain.CodePRMerged, domain.CodeNotAssigned, domain.CodeNoCandidate, domain.CodeReviewersAssigned:
		statusCode = http.StatusConflict
	case domain.CodeUnsupportedMediaType:
		statusCode = http.StatusUnsupportedMediaType
	default:
		statusCode = http.StatusInternalServerError
	}

	h.logger.Error("request error",
		slog.String("code", string(apiErr.Code)),
		slog.String("message", apiErr.Message),
		slog.Int("status", statusCode),
	)

	c.JSON(statusCode, gin.H{
		"error": gin.H{
			"code":    apiErr.Code,
			"message": apiErr.Message,
		},
	})
}

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	r.GET("/health", h.GetHealth)
	r.GET("/stats", h.GetStats)

	r.POST("/team/add", h.TeamAdd)
	r.POST("/team/deactivate", h.TeamDeactivate)
	r.GET("/team/get", h.TeamGet)

	r.POST("/users/setIsActive", h.UsersSetIsActive)
	r.GET("/users/getReview", h.UsersGetReview)

	r.POST("/pullRequest/create", h.PullRequestCreate)
	r.POST("/pullRequest/merge", h.PullRequestMerge)
	r.POST("/pullRequest/reassign", h.PullRequestReassign)
	r.POST("/pullRequest/assign", h.PullRequestAssign)
}
