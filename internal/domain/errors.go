package domain

import "errors"

// Domain errors
var (
	// Team errors
	ErrTeamNotFound = errors.New("team not found")

	// User errors
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")
	ErrInvalidUserStatus = errors.New("invalid user status")
	ErrUserNotActive     = errors.New("user is not active")

	// Pull Request errors
	ErrPRExists                 = errors.New("pull request already exists")
	ErrPRNotFound               = errors.New("pull request not found")
	ErrPRMerged                 = errors.New("pull request is already merged")
	ErrReviewerNotFound         = errors.New("reviewer not assigned to this PR")
	ErrNoAvailableReviewer      = errors.New("no available reviewer found")
	ErrAuthorNotInTeam          = errors.New("author is not in any team")
	ErrInvalidPRStatus          = errors.New("invalid pull request status")
	ErrReviewersAlreadyAssigned = errors.New("reviewers already assigned to this PR")
	ErrReviewerNotInTeam        = errors.New("reviewer is not in author's team")
	ErrAuthorAsReviewer         = errors.New("author cannot be a reviewer")

	// General errors
	ErrInvalidInput      = errors.New("invalid input")
	ErrInternalError     = errors.New("internal server error")
	ErrDatabaseError     = errors.New("database error")
	ErrTransactionFailed = errors.New("transaction failed")
)

// ErrorCode represents API error codes
type ErrorCode string

const (
	CodePRExists             ErrorCode = "PR_EXISTS"
	CodePRMerged             ErrorCode = "PR_MERGED"
	CodeNotAssigned          ErrorCode = "NOT_ASSIGNED"
	CodeNoCandidate          ErrorCode = "NO_CANDIDATE"
	CodeReviewersAssigned    ErrorCode = "REVIEWERS_ASSIGNED"
	CodeNotFound             ErrorCode = "NOT_FOUND"
	CodeBadRequest           ErrorCode = "BAD_REQUEST"
	CodeUnsupportedMediaType ErrorCode = "UNSUPPORTED_MEDIA_TYPE"
	CodeInternalError        ErrorCode = "INTERNAL_ERROR"
)

// APIError represents a structured error response
type APIError struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
}

// Error implements error interface
func (e *APIError) Error() string {
	return string(e.Code) + ": " + e.Message
}

// NewAPIError creates a new API error
func NewAPIError(code ErrorCode, message string) *APIError {
	return &APIError{
		Code:    code,
		Message: message,
	}
}

// ToAPIError converts domain errors to API errors
func ToAPIError(err error) *APIError {
	switch {
	case errors.Is(err, ErrPRExists):
		return NewAPIError(CodePRExists, err.Error())
	case errors.Is(err, ErrPRMerged):
		return NewAPIError(CodePRMerged, err.Error())
	case errors.Is(err, ErrReviewerNotFound):
		return NewAPIError(CodeNotAssigned, err.Error())
	case errors.Is(err, ErrNoAvailableReviewer):
		return NewAPIError(CodeNoCandidate, err.Error())
	case errors.Is(err, ErrReviewersAlreadyAssigned):
		return NewAPIError(CodeReviewersAssigned, err.Error())
	case errors.Is(err, ErrTeamNotFound), errors.Is(err, ErrUserNotFound), errors.Is(err, ErrPRNotFound):
		return NewAPIError(CodeNotFound, err.Error())
	case errors.Is(err, ErrInvalidInput), errors.Is(err, ErrInvalidUserStatus), errors.Is(err, ErrInvalidPRStatus),
		errors.Is(err, ErrUserNotActive), errors.Is(err, ErrReviewerNotInTeam), errors.Is(err, ErrAuthorAsReviewer):
		return NewAPIError(CodeBadRequest, err.Error())
	default:
		return NewAPIError(CodeInternalError, "internal server error")
	}
}
