package handlers

// Common error message constants shared across handlers
const (
	ErrMsgUserNotFound              = "User not found"
	ErrMsgInvalidRequestBody        = "Invalid request body"
	ErrMsgFailedToHashPassword      = "Failed to hash password"
	ErrMsgFailedToCheckAdminStatus  = "Failed to check admin status: "
	ErrMsgFailedToVerifyAdminStatus = "Failed to verify admin status"
	ErrMsgUnauthorized              = "Unauthorized"
	ErrMsgUserIDNotFound            = "User ID not found"
	ErrMsgInvalidAssessmentID       = "Invalid assessment ID"
	ErrMsgPermissionDenied          = "permission denied"
	ErrMsgNotFound                  = "not found"
)

// API path constants
const (
	AuthAPIBasePath = "/api/v1/auth"
)

// Audit action constants
const (
	AuditActionOAuthError = "user.oauth.error"
)
