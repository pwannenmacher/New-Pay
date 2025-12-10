package service

import (
	"fmt"
	"new-pay/internal/models"
	"new-pay/internal/repository"
	"time"
)

// SelfAssessmentService handles business logic for self-assessments
type SelfAssessmentService struct {
	selfAssessmentRepo *repository.SelfAssessmentRepository
	catalogRepo        *repository.CatalogRepository
	auditRepo          *repository.AuditRepository
}

// NewSelfAssessmentService creates a new self-assessment service
func NewSelfAssessmentService(
	selfAssessmentRepo *repository.SelfAssessmentRepository,
	catalogRepo *repository.CatalogRepository,
	auditRepo *repository.AuditRepository,
) *SelfAssessmentService {
	return &SelfAssessmentService{
		selfAssessmentRepo: selfAssessmentRepo,
		catalogRepo:        catalogRepo,
		auditRepo:          auditRepo,
	}
}

// CreateSelfAssessment creates a new self-assessment in draft status
func (s *SelfAssessmentService) CreateSelfAssessment(catalogID uint, userID uint) (*models.SelfAssessment, error) {
	// Verify catalog exists and is in active phase
	catalog, err := s.catalogRepo.GetCatalogByID(catalogID)
	if err != nil {
		return nil, err
	}
	if catalog == nil {
		return nil, fmt.Errorf("catalog not found")
	}
	if catalog.Phase != "active" {
		return nil, fmt.Errorf("can only create self-assessments for active catalogs")
	}

	// Check if catalog is valid for current date
	now := time.Now()
	if now.Before(catalog.ValidFrom) || now.After(catalog.ValidUntil) {
		return nil, fmt.Errorf("catalog is not valid for current date")
	}

	// Check if user already has a self-assessment for this catalog
	existing, err := s.selfAssessmentRepo.GetByCatalogAndUser(catalogID, userID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, fmt.Errorf("self-assessment already exists for this catalog")
	}

	// Create new assessment
	assessment := &models.SelfAssessment{
		CatalogID: catalogID,
		UserID:    userID,
		Status:    "draft",
	}

	if err := s.selfAssessmentRepo.Create(assessment); err != nil {
		return nil, err
	}

	// Audit log
	s.auditRepo.Create(&models.AuditLog{
		UserID:   &userID,
		Action:   "create",
		Resource: "self_assessment",
		Details:  fmt.Sprintf("Created self-assessment (ID: %d) for catalog %d", assessment.ID, catalogID),
	})

	return assessment, nil
}

// GetSelfAssessment retrieves a self-assessment by ID with permission checks
func (s *SelfAssessmentService) GetSelfAssessment(assessmentID uint, userID uint, userRoles []string) (*models.SelfAssessment, error) {
	assessment, err := s.selfAssessmentRepo.GetByID(assessmentID)
	if err != nil {
		return nil, err
	}
	if assessment == nil {
		return nil, fmt.Errorf("self-assessment not found")
	}

	// Permission checks
	isOwner := assessment.UserID == userID
	isReviewer := contains(userRoles, "reviewer")
	isAdmin := contains(userRoles, "admin")

	// Owners can always see their own assessments
	if isOwner {
		return assessment, nil
	}

	// Admins can only see metadata (status, dates) - content filtered in handler
	if isAdmin {
		return assessment, nil
	}

	// Reviewers can only see submitted or later assessments
	if isReviewer && assessment.Status != "draft" && assessment.Status != "closed" {
		return assessment, nil
	}

	return nil, fmt.Errorf("permission denied: cannot view this self-assessment")
}

// GetUserSelfAssessments retrieves all self-assessments for a user
func (s *SelfAssessmentService) GetUserSelfAssessments(userID uint) ([]models.SelfAssessment, error) {
	return s.selfAssessmentRepo.GetByUserID(userID)
}

// GetActiveCatalogs retrieves catalogs that are active and valid for current date
func (s *SelfAssessmentService) GetActiveCatalogs() ([]models.CriteriaCatalog, error) {
	catalogs, err := s.catalogRepo.GetCatalogsByPhase("active")
	if err != nil {
		return nil, err
	}

	// Filter by current date validity
	now := time.Now()
	var validCatalogs []models.CriteriaCatalog
	for _, catalog := range catalogs {
		if !now.Before(catalog.ValidFrom) && !now.After(catalog.ValidUntil) {
			validCatalogs = append(validCatalogs, catalog)
		}
	}

	return validCatalogs, nil
}

// GetVisibleSelfAssessments retrieves self-assessments visible to a user based on role
func (s *SelfAssessmentService) GetVisibleSelfAssessments(userID uint, userRoles []string) ([]models.SelfAssessment, error) {
	isReviewer := contains(userRoles, "reviewer")
	isAdmin := contains(userRoles, "admin")

	if isAdmin {
		// Admins see all metadata
		return s.selfAssessmentRepo.GetAllMetadata()
	}

	if isReviewer {
		// Reviewers see submitted and later assessments
		return s.selfAssessmentRepo.GetVisibleToReviewers()
	}

	// Regular users see only their own
	return s.selfAssessmentRepo.GetByUserID(userID)
}

// UpdateSelfAssessmentStatus transitions a self-assessment to a new status
func (s *SelfAssessmentService) UpdateSelfAssessmentStatus(assessmentID uint, newStatus string, userID uint, userRoles []string) error {
	// Get existing assessment
	assessment, err := s.selfAssessmentRepo.GetByID(assessmentID)
	if err != nil {
		return err
	}
	if assessment == nil {
		return fmt.Errorf("self-assessment not found")
	}

	oldStatus := assessment.Status

	// Check ownership for draft/submitted transitions
	if newStatus == "submitted" && assessment.UserID != userID {
		return fmt.Errorf("permission denied: only the owner can submit their self-assessment")
	}

	// Validate status transitions
	if err := s.validateStatusTransition(oldStatus, newStatus, userRoles); err != nil {
		return err
	}

	// Handle closed status reversal within 24h
	if oldStatus == "closed" && assessment.ClosedAt != nil {
		if time.Since(*assessment.ClosedAt) > 24*time.Hour {
			return fmt.Errorf("cannot revert closed status after 24 hours")
		}
		if assessment.PreviousStatus != nil {
			newStatus = *assessment.PreviousStatus
		}
	}

	// Update timestamps based on new status
	now := time.Now()
	switch newStatus {
	case "submitted":
		assessment.SubmittedAt = &now
	case "in_review":
		assessment.InReviewAt = &now
	case "reviewed":
		assessment.ReviewedAt = &now
	case "discussion":
		assessment.DiscussionStartedAt = &now
	case "archived":
		assessment.ArchivedAt = &now
	case "closed":
		assessment.ClosedAt = &now
		assessment.PreviousStatus = &oldStatus
	}

	assessment.Status = newStatus

	// Update assessment
	if err := s.selfAssessmentRepo.Update(assessment); err != nil {
		return err
	}

	// Audit log
	s.auditRepo.Create(&models.AuditLog{
		UserID:   &userID,
		Action:   "update_status",
		Resource: "self_assessment",
		Details:  fmt.Sprintf("Self-assessment %d status changed: %s -> %s", assessmentID, oldStatus, newStatus),
	})

	return nil
}

// validateStatusTransition validates if a status transition is allowed
func (s *SelfAssessmentService) validateStatusTransition(fromStatus, toStatus string, userRoles []string) error {
	isReviewer := contains(userRoles, "reviewer")
	isAdmin := contains(userRoles, "admin")

	// Define allowed transitions
	allowedTransitions := map[string][]string{
		"draft":      {"submitted", "closed"},
		"submitted":  {"in_review", "closed"},
		"in_review":  {"reviewed", "closed"},
		"reviewed":   {"discussion", "closed"},
		"discussion": {"archived", "closed"},
		"archived":   {},                                                            // Final state
		"closed":     {"draft", "submitted", "in_review", "reviewed", "discussion"}, // Can revert within 24h
	}

	// Check if user has permission for this transition
	if toStatus == "in_review" || toStatus == "reviewed" || toStatus == "discussion" || toStatus == "archived" {
		if !isReviewer && !isAdmin {
			return fmt.Errorf("permission denied: only reviewers can transition to %s status", toStatus)
		}
	}

	allowed, ok := allowedTransitions[fromStatus]
	if !ok {
		return fmt.Errorf("invalid current status: %s", fromStatus)
	}

	for _, validStatus := range allowed {
		if toStatus == validStatus {
			return nil
		}
	}

	return fmt.Errorf("cannot transition from %s to %s status", fromStatus, toStatus)
}
