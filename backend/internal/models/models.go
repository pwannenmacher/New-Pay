package models

import (
	"time"
)

// User represents a user in the system
type User struct {
	ID              uint       `json:"id" db:"id"`
	Email           string     `json:"email" db:"email"`
	PasswordHash    string     `json:"-" db:"password_hash"`
	FirstName       string     `json:"first_name" db:"first_name"`
	LastName        string     `json:"last_name" db:"last_name"`
	EmailVerified   bool       `json:"email_verified" db:"email_verified"`
	EmailVerifiedAt *time.Time `json:"email_verified_at,omitempty" db:"email_verified_at"`
	IsActive        bool       `json:"is_active" db:"is_active"`
	LastLoginAt     *time.Time `json:"last_login_at,omitempty" db:"last_login_at"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at" db:"updated_at"`
	OAuthProvider   *string    `json:"oauth_provider,omitempty" db:"oauth_provider"`
	OAuthProviderID *string    `json:"-" db:"oauth_provider_id"`
}

// Role represents a user role
type Role struct {
	ID          uint      `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// Permission represents a permission in the system
type Permission struct {
	ID          uint      `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Resource    string    `json:"resource" db:"resource"`
	Action      string    `json:"action" db:"action"`
	Description string    `json:"description" db:"description"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// UserRole represents the many-to-many relationship between users and roles
type UserRole struct {
	UserID    uint      `json:"user_id" db:"user_id"`
	RoleID    uint      `json:"role_id" db:"role_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// RolePermission represents the many-to-many relationship between roles and permissions
type RolePermission struct {
	RoleID       uint      `json:"role_id" db:"role_id"`
	PermissionID uint      `json:"permission_id" db:"permission_id"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// EmailVerificationToken represents a token for email verification
type EmailVerificationToken struct {
	ID        uint       `json:"id" db:"id"`
	UserID    uint       `json:"user_id" db:"user_id"`
	Token     string     `json:"token" db:"token"`
	ExpiresAt time.Time  `json:"expires_at" db:"expires_at"`
	UsedAt    *time.Time `json:"used_at,omitempty" db:"used_at"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
}

// PasswordResetToken represents a token for password reset
type PasswordResetToken struct {
	ID        uint       `json:"id" db:"id"`
	UserID    uint       `json:"user_id" db:"user_id"`
	Token     string     `json:"token" db:"token"`
	ExpiresAt time.Time  `json:"expires_at" db:"expires_at"`
	UsedAt    *time.Time `json:"used_at,omitempty" db:"used_at"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
}

// Session represents a user session
type Session struct {
	ID             string    `json:"id" db:"id"`
	UserID         uint      `json:"user_id" db:"user_id"`
	SessionID      string    `json:"session_id" db:"session_id"` // Groups access and refresh tokens from same login
	JTI            string    `json:"jti" db:"jti"`
	TokenType      string    `json:"token_type" db:"token_type"` // "access" or "refresh"
	ExpiresAt      time.Time `json:"expires_at" db:"expires_at"`
	LastActivityAt time.Time `json:"last_activity_at" db:"last_activity_at"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	IPAddress      string    `json:"ip_address,omitempty" db:"ip_address"`
	UserAgent      string    `json:"user_agent,omitempty" db:"user_agent"`
}

// AuditLog represents an audit log entry
type AuditLog struct {
	ID        uint      `json:"id" db:"id"`
	UserID    *uint     `json:"user_id,omitempty" db:"user_id"`
	UserEmail *string   `json:"user_email,omitempty" db:"user_email"`
	Action    string    `json:"action" db:"action"`
	Resource  string    `json:"resource" db:"resource"`
	Details   string    `json:"details,omitempty" db:"details"`
	IPAddress string    `json:"ip_address,omitempty" db:"ip_address"`
	UserAgent string    `json:"user_agent,omitempty" db:"user_agent"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// UserWithRoles extends User with roles information
type UserWithRoles struct {
	User
	Roles []Role `json:"roles"`
}

// OAuthConnection represents a connection between a user and an OAuth provider
type OAuthConnection struct {
	ID         uint      `json:"id" db:"id"`
	UserID     uint      `json:"user_id" db:"user_id"`
	Provider   string    `json:"provider" db:"provider"`
	ProviderID string    `json:"provider_id" db:"provider_id"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}

// CriteriaCatalog represents a criteria catalog with phases and validity
type CriteriaCatalog struct {
	ID          uint       `json:"id" db:"id"`
	Name        string     `json:"name" db:"name"`
	Description *string    `json:"description,omitempty" db:"description"`
	ValidFrom   time.Time  `json:"valid_from" db:"valid_from"`
	ValidUntil  time.Time  `json:"valid_until" db:"valid_until"`
	Phase       string     `json:"phase" db:"phase"` // draft, review, archived
	CreatedBy   *uint      `json:"created_by,omitempty" db:"created_by"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
	PublishedAt *time.Time `json:"published_at,omitempty" db:"published_at"`
	ArchivedAt  *time.Time `json:"archived_at,omitempty" db:"archived_at"`
}

// Category represents a category within a criteria catalog
type Category struct {
	ID          uint      `json:"id" db:"id"`
	CatalogID   uint      `json:"catalog_id" db:"catalog_id"`
	Name        string    `json:"name" db:"name"`
	Description *string   `json:"description,omitempty" db:"description"`
	SortOrder   int       `json:"sort_order" db:"sort_order"`
	Weight      *float64  `json:"weight,omitempty" db:"weight"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// Level represents a level (column) in the criteria matrix
type Level struct {
	ID          uint      `json:"id" db:"id"`
	CatalogID   uint      `json:"catalog_id" db:"catalog_id"`
	Name        string    `json:"name" db:"name"`
	LevelNumber int       `json:"level_number" db:"level_number"`
	Description *string   `json:"description,omitempty" db:"description"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// Path represents a path (row) within a category
type Path struct {
	ID          uint      `json:"id" db:"id"`
	CategoryID  uint      `json:"category_id" db:"category_id"`
	Name        string    `json:"name" db:"name"`
	Description *string   `json:"description,omitempty" db:"description"`
	SortOrder   int       `json:"sort_order" db:"sort_order"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// PathLevelDescription represents a cell in the criteria matrix
type PathLevelDescription struct {
	ID          uint      `json:"id" db:"id"`
	PathID      uint      `json:"path_id" db:"path_id"`
	LevelID     uint      `json:"level_id" db:"level_id"`
	Description string    `json:"description" db:"description"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// CatalogChange represents a change log entry for catalog modifications
type CatalogChange struct {
	ID         uint      `json:"id" db:"id"`
	CatalogID  uint      `json:"catalog_id" db:"catalog_id"`
	EntityType string    `json:"entity_type" db:"entity_type"` // catalog, category, path, level, description
	EntityID   uint      `json:"entity_id" db:"entity_id"`
	FieldName  string    `json:"field_name" db:"field_name"`
	OldValue   *string   `json:"old_value,omitempty" db:"old_value"`
	NewValue   *string   `json:"new_value,omitempty" db:"new_value"`
	ChangedBy  *uint     `json:"changed_by,omitempty" db:"changed_by"`
	ChangedAt  time.Time `json:"changed_at" db:"changed_at"`
}

// CatalogWithDetails extends CriteriaCatalog with nested structures
type CatalogWithDetails struct {
	CriteriaCatalog
	Categories []CategoryWithPaths `json:"categories,omitempty"`
	Levels     []Level             `json:"levels,omitempty"`
}

// CategoryWithPaths extends Category with paths
type CategoryWithPaths struct {
	Category
	Paths []PathWithDescriptions `json:"paths,omitempty"`
}

// PathWithDescriptions extends Path with descriptions for all levels
type PathWithDescriptions struct {
	Path
	Descriptions []PathLevelDescription `json:"descriptions,omitempty"`
}

// SelfAssessment represents a user's self-assessment for a catalog
type SelfAssessment struct {
	ID                    uint       `json:"id" db:"id"`
	CatalogID             uint       `json:"catalog_id" db:"catalog_id"`
	UserID                uint       `json:"user_id" db:"user_id"`
	Status                string     `json:"status" db:"status"` // draft, submitted, in_review, review_consolidation, reviewed, discussion, archived, closed
	CreatedAt             time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at" db:"updated_at"`
	SubmittedAt           *time.Time `json:"submitted_at,omitempty" db:"submitted_at"`
	InReviewAt            *time.Time `json:"in_review_at,omitempty" db:"in_review_at"`
	ReviewConsolidationAt *time.Time `json:"review_consolidation_at,omitempty" db:"review_consolidation_at"`
	ReviewedAt            *time.Time `json:"reviewed_at,omitempty" db:"reviewed_at"`
	DiscussionStartedAt   *time.Time `json:"discussion_started_at,omitempty" db:"discussion_started_at"`
	ArchivedAt            *time.Time `json:"archived_at,omitempty" db:"archived_at"`
	ClosedAt              *time.Time `json:"closed_at,omitempty" db:"closed_at"`
	PreviousStatus        *string    `json:"previous_status,omitempty" db:"previous_status"`
}

// SelfAssessmentWithDetails includes user and catalog information
type SelfAssessmentWithDetails struct {
	SelfAssessment
	UserEmail        string `json:"user_email,omitempty"`
	UserName         string `json:"user_name,omitempty"` // first_name + last_name
	CatalogName      string `json:"catalog_name,omitempty"`
	ReviewsStarted   int    `json:"reviews_started,omitempty"`   // Number of reviewers who have started (at least one category)
	ReviewsCompleted int    `json:"reviews_completed,omitempty"` // Number of reviewers who have completed all categories
}

// AssessmentResponse represents a user's selection for one category
type AssessmentResponse struct {
	ID                       uint      `json:"id" db:"id"`
	AssessmentID             uint      `json:"assessment_id" db:"assessment_id"`
	CategoryID               uint      `json:"category_id" db:"category_id"`
	PathID                   uint      `json:"path_id" db:"path_id"`
	LevelID                  uint      `json:"level_id" db:"level_id"`
	Justification            string    `json:"justification" db:"justification"`                                     // Decrypted justification (not stored)
	EncryptedJustificationID *int64    `json:"encrypted_justification_id,omitempty" db:"encrypted_justification_id"` // Reference to encrypted_records
	CreatedAt                time.Time `json:"created_at" db:"created_at"`
	UpdatedAt                time.Time `json:"updated_at" db:"updated_at"`
}

// AssessmentResponseWithDetails includes category, path, and level information
type AssessmentResponseWithDetails struct {
	AssessmentResponse
	CategoryName         string  `json:"category_name"`
	CategorySortOrder    int     `json:"category_sort_order"`
	PathName             string  `json:"path_name"`
	PathDescription      *string `json:"path_description,omitempty"`
	LevelName            string  `json:"level_name"`
	LevelNumber          int     `json:"level_number"`
	LevelDescription     *string `json:"level_description,omitempty"`
	PathLevelDescription string  `json:"path_level_description"` // The description of the path-level combination
}

// AssessmentCompleteness represents the completion status of a self-assessment
type AssessmentCompleteness struct {
	TotalCategories     int     `json:"total_categories"`
	CompletedCategories int     `json:"completed_categories"`
	PercentComplete     float64 `json:"percent_complete"`
	IsComplete          bool    `json:"is_complete"`
	MissingCategories   []uint  `json:"missing_categories,omitempty"`
}

// WeightedScore represents the calculated weighted average score for a self-assessment
type WeightedScore struct {
	WeightedAverage float64 `json:"weighted_average"` // The calculated weighted score
	OverallLevel    string  `json:"overall_level"`    // The corresponding level letter (A, B, C, etc.)
	LevelNumber     int     `json:"level_number"`     // The corresponding level number
	IsComplete      bool    `json:"is_complete"`      // Whether all categories have responses
}

// ReviewerResponse represents a reviewer's assessment of one category in a self-assessment
type ReviewerResponse struct {
	ID                       uint      `json:"id" db:"id"`
	AssessmentID             uint      `json:"assessment_id" db:"assessment_id"`
	CategoryID               uint      `json:"category_id" db:"category_id"`
	ReviewerUserID           uint      `json:"reviewer_user_id" db:"reviewer_user_id"`
	PathID                   uint      `json:"path_id" db:"path_id"`
	LevelID                  uint      `json:"level_id" db:"level_id"`
	Justification            string    `json:"justification" db:"-"`                                                 // Decrypted justification (not stored in DB)
	EncryptedJustificationID *int64    `json:"encrypted_justification_id,omitempty" db:"encrypted_justification_id"` // Reference to encrypted_records
	CreatedAt                time.Time `json:"created_at" db:"created_at"`
	UpdatedAt                time.Time `json:"updated_at" db:"updated_at"`
}

// ReviewCompletionStatus represents the status of reviews for an assessment
type ReviewCompletionStatus struct {
	TotalReviewers               int                      `json:"total_reviewers"`
	CompleteReviews              int                      `json:"complete_reviews"`
	CanConsolidate               bool                     `json:"can_consolidate"`
	ReviewersWithCompleteReviews []ReviewerCompletionInfo `json:"reviewers_with_complete_reviews"`
}

// ReviewerCompletionInfo contains info about a reviewer who completed their review
type ReviewerCompletionInfo struct {
	ReviewerID   uint      `json:"reviewer_id"`
	ReviewerName string    `json:"reviewer_name"`
	CompletedAt  time.Time `json:"completed_at"`
}

// ConsolidationOverride represents a manually adjusted value during review consolidation
type ConsolidationOverride struct {
	ID                       uint                            `json:"id" db:"id"`
	AssessmentID             uint                            `json:"assessment_id" db:"assessment_id"`
	CategoryID               uint                            `json:"category_id" db:"category_id"`
	PathID                   uint                            `json:"path_id" db:"path_id"`
	LevelID                  uint                            `json:"level_id" db:"level_id"`
	Justification            string                          `json:"justification" db:"-"`                                                 // Decrypted justification (not stored in DB)
	EncryptedJustificationID *int64                          `json:"encrypted_justification_id,omitempty" db:"encrypted_justification_id"` // Reference to encrypted_records
	CreatedByUserID          uint                            `json:"created_by_user_id" db:"created_by_user_id"`
	CreatedAt                time.Time                       `json:"created_at" db:"created_at"`
	UpdatedAt                time.Time                       `json:"updated_at" db:"updated_at"`
	Approvals                []ConsolidationOverrideApproval `json:"approvals,omitempty" db:"-"` // Loaded separately
	ApprovalCount            int                             `json:"approval_count" db:"-"`      // Number of approvals
	IsApproved               bool                            `json:"is_approved" db:"-"`         // Has at least one approval
}

// ConsolidationOverrideApproval tracks reviewer approvals for overrides
type ConsolidationOverrideApproval struct {
	ID               uint      `json:"id" db:"id"`
	OverrideID       uint      `json:"override_id" db:"override_id"`
	ApprovedByUserID uint      `json:"approved_by_user_id" db:"approved_by_user_id"`
	ApprovedByName   string    `json:"approved_by_name,omitempty" db:"-"` // Loaded separately
	ApprovedAt       time.Time `json:"approved_at" db:"approved_at"`
}

// ConsolidationAveragedApproval tracks reviewer approvals for averaged responses (without override)
type ConsolidationAveragedApproval struct {
	ID               uint      `json:"id" db:"id"`
	AssessmentID     uint      `json:"assessment_id" db:"assessment_id"`
	CategoryID       uint      `json:"category_id" db:"category_id"`
	ApprovedByUserID uint      `json:"approved_by_user_id" db:"approved_by_user_id"`
	ApprovedByName   string    `json:"approved_by_name,omitempty" db:"-"` // Loaded separately
	ApprovedAt       time.Time `json:"approved_at" db:"approved_at"`
}

// AveragedReviewerResponse represents aggregated reviewer responses for one category
type AveragedReviewerResponse struct {
	CategoryID             uint                            `json:"category_id"`
	CategoryName           string                          `json:"category_name"`
	CategorySortOrder      int                             `json:"category_sort_order"`
	AverageLevelNumber     float64                         `json:"average_level_number"`
	AverageLevelName       string                          `json:"average_level_name"` // Computed from average
	ReviewerCount          int                             `json:"reviewer_count"`
	ReviewerJustifications []string                        `json:"reviewer_justifications,omitempty"` // All reviewer justifications for this category
	Approvals              []ConsolidationAveragedApproval `json:"approvals,omitempty" db:"-"`        // Loaded separately
	ApprovalCount          int                             `json:"approval_count" db:"-"`             // Number of approvals
	IsApproved             bool                            `json:"is_approved" db:"-"`                // Has at least 2 approvals
}

// FinalConsolidation represents the final consolidation comment
type FinalConsolidation struct {
	ID                 uint                         `json:"id" db:"id"`
	AssessmentID       uint                         `json:"assessment_id" db:"assessment_id"`
	Comment            string                       `json:"comment" db:"-"`                                           // Decrypted comment (not stored in DB)
	EncryptedCommentID *int64                       `json:"encrypted_comment_id,omitempty" db:"encrypted_comment_id"` // Reference to encrypted_records
	CreatedByUserID    uint                         `json:"created_by_user_id" db:"created_by_user_id"`
	CreatedAt          time.Time                    `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time                    `json:"updated_at" db:"updated_at"`
	Approvals          []FinalConsolidationApproval `json:"approvals,omitempty" db:"-"` // Loaded separately
	ApprovalCount      int                          `json:"approval_count" db:"-"`      // Number of approvals
	RequiredApprovals  int                          `json:"required_approvals" db:"-"`  // How many approvals needed (= reviewer count)
	IsFullyApproved    bool                         `json:"is_fully_approved" db:"-"`   // All required approvals received
}

// FinalConsolidationApproval tracks reviewer approvals for final consolidation
type FinalConsolidationApproval struct {
	ID               uint      `json:"id" db:"id"`
	AssessmentID     uint      `json:"assessment_id" db:"assessment_id"`
	ApprovedByUserID uint      `json:"approved_by_user_id" db:"approved_by_user_id"`
	ApprovedByName   string    `json:"approved_by_name,omitempty" db:"-"` // Loaded separately
	ApprovedAt       time.Time `json:"approved_at" db:"approved_at"`
}

// CategoryDiscussionComment represents category-specific public comments for discussion
type CategoryDiscussionComment struct {
	ID                 uint      `json:"id" db:"id"`
	AssessmentID       uint      `json:"assessment_id" db:"assessment_id"`
	CategoryID         uint      `json:"category_id" db:"category_id"`
	Comment            string    `json:"comment" db:"-"`                                           // Decrypted comment
	EncryptedCommentID *int64    `json:"encrypted_comment_id,omitempty" db:"encrypted_comment_id"` // Reference to encrypted_records
	CreatedByUserID    uint      `json:"created_by_user_id" db:"created_by_user_id"`
	CreatedAt          time.Time `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time `json:"updated_at" db:"updated_at"`
}

// DiscussionConfirmation represents a confirmation that the discussion meeting took place
type DiscussionConfirmation struct {
	ID           uint      `json:"id" db:"id"`
	AssessmentID uint      `json:"assessment_id" db:"assessment_id"`
	UserID       uint      `json:"user_id" db:"user_id"`
	UserType     string    `json:"user_type" db:"user_type"` // "reviewer" or "owner"
	ConfirmedAt  time.Time `json:"confirmed_at" db:"confirmed_at"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`

	// Populated fields (not from DB)
	UserName  string `json:"user_name,omitempty" db:"-"`
	UserEmail string `json:"user_email,omitempty" db:"-"`
}

// ConsolidationData contains all data needed for the consolidation page
type ConsolidationData struct {
	Assessment                 SelfAssessment                  `json:"assessment"`
	UserResponses              []AssessmentResponseWithDetails `json:"user_responses"`
	AveragedResponses          []AveragedReviewerResponse      `json:"averaged_responses"`
	Overrides                  []ConsolidationOverride         `json:"overrides"`
	Catalog                    CatalogWithDetails              `json:"catalog"`
	CurrentUserResponses       []ReviewerResponse              `json:"current_user_responses"`        // Current reviewer's own responses
	FinalConsolidation         *FinalConsolidation             `json:"final_consolidation,omitempty"` // Final consolidation if exists
	CategoryDiscussionComments []CategoryDiscussionComment     `json:"category_discussion_comments"`  // Public category comments
	AllCategoriesApproved      bool                            `json:"all_categories_approved"`       // True if all categories have required approvals
}

// DiscussionResult represents the frozen discussion data
type DiscussionResult struct {
	ID                       uint                       `json:"id" db:"id"`
	AssessmentID             uint                       `json:"assessment_id" db:"assessment_id"`
	AssessmentStatus         string                     `json:"assessment_status,omitempty" db:"-"` // Populated from assessment
	WeightedOverallLevelNum  float64                    `json:"weighted_overall_level_number" db:"weighted_overall_level_number"`
	WeightedOverallLevelID   uint                       `json:"weighted_overall_level_id" db:"weighted_overall_level_id"`
	WeightedOverallLevelName string                     `json:"weighted_overall_level_name,omitempty" db:"-"` // Loaded separately
	FinalComment             string                     `json:"final_comment" db:"-"`                         // Decrypted
	FinalCommentEncrypted    []byte                     `json:"-" db:"final_comment_encrypted"`
	FinalCommentNonce        []byte                     `json:"-" db:"final_comment_nonce"`
	DiscussionNote           *string                    `json:"discussion_note,omitempty" db:"discussion_note"`
	UserApprovedAt           *time.Time                 `json:"user_approved_at,omitempty" db:"user_approved_at"`
	CreatedAt                time.Time                  `json:"created_at" db:"created_at"`
	UpdatedAt                time.Time                  `json:"updated_at" db:"updated_at"`
	CategoryResults          []DiscussionCategoryResult `json:"category_results,omitempty" db:"-"`
	Reviewers                []DiscussionReviewer       `json:"reviewers,omitempty" db:"-"`
	Confirmations            []DiscussionConfirmation   `json:"confirmations,omitempty" db:"-"` // Discussion confirmations
}

// DiscussionCategoryResult represents per-category results in discussion
type DiscussionCategoryResult struct {
	ID                     uint    `json:"id" db:"id"`
	DiscussionResultID     uint    `json:"discussion_result_id" db:"discussion_result_id"`
	CategoryID             uint    `json:"category_id" db:"category_id"`
	CategoryName           string  `json:"category_name,omitempty" db:"-"`
	UserLevelID            *uint   `json:"user_level_id,omitempty" db:"user_level_id"`
	UserLevelName          string  `json:"user_level_name,omitempty" db:"-"`
	ReviewerLevelID        uint    `json:"reviewer_level_id" db:"reviewer_level_id"`
	ReviewerLevelName      string  `json:"reviewer_level_name,omitempty" db:"-"`
	ReviewerLevelNumber    float64 `json:"reviewer_level_number" db:"reviewer_level_number"`
	JustificationEncrypted []byte  `json:"-" db:"justification_encrypted"`
	JustificationNonce     []byte  `json:"-" db:"justification_nonce"`
	JustificationPlain     *string `json:"justification,omitempty" db:"justification_plain"` // Combined or decrypted
	IsOverride             bool    `json:"is_override" db:"is_override"`
}

// DiscussionReviewer represents reviewers who participated
type DiscussionReviewer struct {
	ID                 uint   `json:"id" db:"id"`
	DiscussionResultID uint   `json:"discussion_result_id" db:"discussion_result_id"`
	ReviewerUserID     uint   `json:"reviewer_user_id" db:"reviewer_user_id"`
	ReviewerName       string `json:"reviewer_name" db:"reviewer_name"`
}
