package service

import (
	"fmt"
	"new-pay/internal/email"
	"new-pay/internal/models"
	"new-pay/internal/repository"
	"time"
)

// CatalogService handles business logic for criteria catalogs
type CatalogService struct {
	catalogRepo        *repository.CatalogRepository
	selfAssessmentRepo *repository.SelfAssessmentRepository
	auditRepo          *repository.AuditRepository
	emailService       *email.Service
}

// NewCatalogService creates a new catalog service
func NewCatalogService(catalogRepo *repository.CatalogRepository, selfAssessmentRepo *repository.SelfAssessmentRepository, auditRepo *repository.AuditRepository, emailService *email.Service) *CatalogService {
	return &CatalogService{
		catalogRepo:        catalogRepo,
		selfAssessmentRepo: selfAssessmentRepo,
		auditRepo:          auditRepo,
		emailService:       emailService,
	}
}

// CreateCatalog creates a new catalog in draft phase
func (s *CatalogService) CreateCatalog(catalog *models.CriteriaCatalog, userID uint) error {
	// Validate dates
	if catalog.ValidFrom.After(catalog.ValidUntil) || catalog.ValidFrom.Equal(catalog.ValidUntil) {
		return fmt.Errorf("valid_from must be before valid_until")
	}

	// Check for overlapping catalogs
	overlaps, err := s.catalogRepo.CheckOverlappingCatalogs(catalog.ValidFrom, catalog.ValidUntil, nil)
	if err != nil {
		return fmt.Errorf("failed to check overlapping catalogs: %w", err)
	}
	if overlaps {
		return fmt.Errorf("catalog validity period overlaps with existing non-archived catalog")
	}

	// Set defaults
	catalog.Phase = "draft"
	catalog.CreatedBy = &userID

	if err := s.catalogRepo.CreateCatalog(catalog); err != nil {
		return err
	}

	// Audit log
	s.auditRepo.Create(&models.AuditLog{
		UserID:   &userID,
		Action:   "create",
		Resource: "catalog",
		Details:  fmt.Sprintf("Created catalog: %s (ID: %d)", catalog.Name, catalog.ID),
	})

	return nil
}

// GetCatalogByID retrieves a catalog by ID
func (s *CatalogService) GetCatalogByID(id uint) (*models.CriteriaCatalog, error) {
	return s.catalogRepo.GetCatalogByID(id)
}

// GetAllCatalogs retrieves all catalogs
func (s *CatalogService) GetAllCatalogs() ([]models.CriteriaCatalog, error) {
	return s.catalogRepo.GetAllCatalogs()
}

// GetCatalogsByPhase retrieves catalogs by phase
func (s *CatalogService) GetCatalogsByPhase(phase string) ([]models.CriteriaCatalog, error) {
	if phase != "draft" && phase != "active" && phase != "archived" {
		return nil, fmt.Errorf("invalid phase: %s", phase)
	}
	return s.catalogRepo.GetCatalogsByPhase(phase)
}

// GetVisibleCatalogs retrieves catalogs visible to a user based on their role
func (s *CatalogService) GetVisibleCatalogs(userRoles []string) ([]models.CriteriaCatalog, error) {
	isAdmin := contains(userRoles, "admin")
	isReviewer := contains(userRoles, "reviewer")

	if isAdmin {
		// Admins see everything
		return s.catalogRepo.GetAllCatalogs()
	}

	if isReviewer {
		// Reviewers see active and archived catalogs
		activeCatalogs, err := s.catalogRepo.GetCatalogsByPhase("active")
		if err != nil {
			return nil, err
		}
		archivedCatalogs, err := s.catalogRepo.GetCatalogsByPhase("archived")
		if err != nil {
			return nil, err
		}
		return append(activeCatalogs, archivedCatalogs...), nil
	}

	// Regular users only see active phase catalogs
	return s.catalogRepo.GetCatalogsByPhase("active")
}

// UpdateCatalog updates a catalog
func (s *CatalogService) UpdateCatalog(catalog *models.CriteriaCatalog, userID uint, userRoles []string) error {
	// Get existing catalog
	existing, err := s.catalogRepo.GetCatalogByID(catalog.ID)
	if err != nil {
		return err
	}
	if existing == nil {
		return fmt.Errorf("catalog not found")
	}

	// Check permissions
	if !canEditCatalog(existing.Phase, userRoles) {
		// Special case: Admins can update valid_until for active catalogs
		if existing.Phase == "active" && contains(userRoles, "admin") {
			// Check if ONLY valid_until is being changed (compare dates only, not timestamps)
			validFromSame := catalog.ValidFrom.Truncate(24 * time.Hour).Equal(existing.ValidFrom.Truncate(24 * time.Hour))
			descriptionSame := compareStringPointers(catalog.Description, existing.Description)

			if catalog.Name != existing.Name ||
				!descriptionSame ||
				!validFromSame ||
				catalog.Phase != existing.Phase {
				return fmt.Errorf("permission denied: can only change valid_until for active catalogs")
			}
			// Validate new valid_until is in the future
			today := time.Now().Truncate(24 * time.Hour)
			newValidUntilDate := catalog.ValidUntil.Truncate(24 * time.Hour)
			if !newValidUntilDate.After(today) {
				return fmt.Errorf("new end date must be after today")
			}
			// Allow this change to proceed
		} else {
			return fmt.Errorf("permission denied: cannot edit catalog in %s phase", existing.Phase)
		}
	}

	// Handle phase transition
	if catalog.Phase != "" && catalog.Phase != existing.Phase {
		if err := s.validatePhaseTransition(catalog.ID, existing.Phase, catalog.Phase, userRoles); err != nil {
			return err
		}
	} else {
		// Keep existing phase if not changing
		catalog.Phase = existing.Phase
	}

	// Validate dates
	if catalog.ValidFrom.After(catalog.ValidUntil) || catalog.ValidFrom.Equal(catalog.ValidUntil) {
		return fmt.Errorf("valid_from must be before valid_until")
	}

	// Check for overlapping catalogs (excluding current catalog)
	overlaps, err := s.catalogRepo.CheckOverlappingCatalogs(catalog.ValidFrom, catalog.ValidUntil, &catalog.ID)
	if err != nil {
		return fmt.Errorf("failed to check overlapping catalogs: %w", err)
	}
	if overlaps {
		return fmt.Errorf("catalog validity period overlaps with existing non-archived catalog")
	}

	// Log changes if in active phase
	if existing.Phase == "active" {
		if err := s.logCatalogChanges(existing, catalog, userID); err != nil {
			return fmt.Errorf("failed to log changes: %w", err)
		}
	}

	if err := s.catalogRepo.UpdateCatalog(catalog); err != nil {
		return err
	}

	// Audit log
	details := fmt.Sprintf("Updated catalog: %s (ID: %d)", catalog.Name, catalog.ID)
	if existing.Phase != catalog.Phase {
		details = fmt.Sprintf("Updated catalog: %s (ID: %d), Phase: %s -> %s", catalog.Name, catalog.ID, existing.Phase, catalog.Phase)
	}
	s.auditRepo.Create(&models.AuditLog{
		UserID:   &userID,
		Action:   "update",
		Resource: "catalog",
		Details:  details,
	})

	return nil
}

// UpdateCatalogValidUntil updates only the valid_until date (shortening runtime)
func (s *CatalogService) UpdateCatalogValidUntil(catalogID uint, newValidUntilStr string) (*models.CriteriaCatalog, error) {
	// Get existing catalog
	catalog, err := s.catalogRepo.GetCatalogByID(catalogID)
	if err != nil {
		return nil, err
	}
	if catalog == nil {
		return nil, fmt.Errorf("catalog not found")
	}

	// Only active catalogs can have their validity shortened
	if catalog.Phase != "active" {
		return nil, fmt.Errorf("only active catalogs can have their validity date changed")
	}

	// Parse new date
	newValidUntil, err := time.Parse("2006-01-02", newValidUntilStr)
	if err != nil {
		return nil, fmt.Errorf("invalid date format, use YYYY-MM-DD")
	}

	// Validate: new date must be after today
	today := time.Now().Truncate(24 * time.Hour)
	if !newValidUntil.After(today) {
		return nil, fmt.Errorf("new end date must be after today")
	}

	// Validate: can only shorten, not extend
	if newValidUntil.After(catalog.ValidUntil) || newValidUntil.Equal(catalog.ValidUntil) {
		return nil, fmt.Errorf("can only shorten the validity period, not extend it")
	}

	// Validate: new date must be after valid_from
	if !newValidUntil.After(catalog.ValidFrom) {
		return nil, fmt.Errorf("end date must be after start date")
	}

	oldValidUntil := catalog.ValidUntil
	catalog.ValidUntil = newValidUntil

	// Update only valid_until in database (bypass normal update checks)
	if err := s.catalogRepo.UpdateCatalogValidUntil(catalogID, newValidUntil); err != nil {
		return nil, fmt.Errorf("failed to update catalog: %w", err)
	}

	// Reload catalog to get updated data
	updatedCatalog, err := s.catalogRepo.GetCatalogByID(catalogID)
	if err != nil {
		return nil, fmt.Errorf("failed to reload catalog: %w", err)
	}

	// Get all open or in-review self-assessments for this catalog
	assessments, err := s.selfAssessmentRepo.GetByCatalogID(catalogID)
	if err != nil {
		return nil, fmt.Errorf("failed to get assessments: %w", err)
	}

	// Send notifications to affected users
	affectedStatuses := []string{"draft", "submitted", "in_review", "reviewed", "discussion"}
	oldDateStr := oldValidUntil.Format("02.01.2006")
	newDateStr := newValidUntil.Format("02.01.2006")

	for _, assessment := range assessments {
		if contains(affectedStatuses, assessment.Status) && assessment.UserEmail != "" {
			// Send email asynchronously
			go func(email, catalogName, oldDate, newDate string) {
				if err := s.emailService.SendCatalogValidityChangeNotification(email, catalogName, oldDate, newDate); err != nil {
					fmt.Printf("Failed to send validity change notification to %s: %v\n", email, err)
				}
			}(assessment.UserEmail, catalog.Name, oldDateStr, newDateStr)
		}
	}

	return updatedCatalog, nil
}

// TransitionToActive transitions a catalog from draft to active phase
func (s *CatalogService) TransitionToActive(catalogID uint, userRoles []string) error {
	if !contains(userRoles, "admin") {
		return fmt.Errorf("permission denied: only admins can transition to active phase")
	}

	catalog, err := s.catalogRepo.GetCatalogByID(catalogID)
	if err != nil {
		return err
	}
	if catalog == nil {
		return fmt.Errorf("catalog not found")
	}

	if catalog.Phase != "draft" {
		return fmt.Errorf("can only transition from draft to active phase")
	}

	// Validate catalog completeness
	if err := s.validateCatalogCompleteness(catalogID); err != nil {
		return fmt.Errorf("catalog validation failed: %w", err)
	}

	return s.catalogRepo.UpdateCatalogPhase(catalogID, "active")
}

// TransitionToArchived transitions a catalog from active to archived phase
func (s *CatalogService) TransitionToArchived(catalogID uint, userRoles []string) error {
	if !contains(userRoles, "admin") {
		return fmt.Errorf("permission denied: only admins can transition to archived phase")
	}

	catalog, err := s.catalogRepo.GetCatalogByID(catalogID)
	if err != nil {
		return err
	}
	if catalog == nil {
		return fmt.Errorf("catalog not found")
	}

	if catalog.Phase != "active" {
		return fmt.Errorf("can only transition from active to archived phase")
	}

	return s.catalogRepo.UpdateCatalogPhase(catalogID, "archived")
}

// DeleteCatalog deletes a catalog (only allowed in draft phase)
func (s *CatalogService) DeleteCatalog(catalogID uint, userID uint, userRoles []string) error {
	if !contains(userRoles, "admin") {
		return fmt.Errorf("permission denied: only admins can delete catalogs")
	}

	catalog, err := s.catalogRepo.GetCatalogByID(catalogID)
	if err != nil {
		return err
	}
	if catalog == nil {
		return fmt.Errorf("catalog not found")
	}

	if catalog.Phase != "draft" {
		return fmt.Errorf("can only delete catalogs in draft phase")
	}

	if err := s.catalogRepo.DeleteCatalog(catalogID); err != nil {
		return err
	}

	// Audit log
	s.auditRepo.Create(&models.AuditLog{
		UserID:   &userID,
		Action:   "delete",
		Resource: "catalog",
		Details:  fmt.Sprintf("Deleted catalog: %s (ID: %d)", catalog.Name, catalogID),
	})

	return nil
}

// GetCatalogWithDetails retrieves a catalog with all nested entities
func (s *CatalogService) GetCatalogWithDetails(catalogID uint, userRoles []string) (*models.CatalogWithDetails, error) {
	catalog, err := s.catalogRepo.GetCatalogByID(catalogID)
	if err != nil {
		return nil, err
	}
	if catalog == nil {
		return nil, fmt.Errorf("catalog not found")
	}

	// Check permissions
	if !canViewCatalog(catalog.Phase, userRoles) {
		return nil, fmt.Errorf("permission denied: cannot view catalog in %s phase", catalog.Phase)
	}

	return s.catalogRepo.GetCatalogWithDetails(catalogID)
}

// CreateCategory creates a new category
func (s *CatalogService) CreateCategory(category *models.Category, userID uint, userRoles []string) error {
	catalog, err := s.catalogRepo.GetCatalogByID(category.CatalogID)
	if err != nil {
		return err
	}
	if catalog == nil {
		return fmt.Errorf("catalog not found")
	}

	if !canEditStructure(catalog.Phase, userRoles) {
		return fmt.Errorf("permission denied: cannot add categories in %s phase", catalog.Phase)
	}

	if err := s.catalogRepo.CreateCategory(category); err != nil {
		return err
	}

	// Audit log
	s.auditRepo.Create(&models.AuditLog{
		UserID:   &userID,
		Action:   "create",
		Resource: "category",
		Details:  fmt.Sprintf("Created category: %s (ID: %d) in catalog %d", category.Name, category.ID, category.CatalogID),
	})

	return nil
}

// UpdateCategory updates a category
func (s *CatalogService) UpdateCategory(category *models.Category, userID uint, userRoles []string) error {
	// Get the catalog to check permissions
	categories, err := s.catalogRepo.GetCategoriesByCatalogID(category.CatalogID)
	if err != nil {
		return err
	}

	var oldCategory *models.Category
	for _, c := range categories {
		if c.ID == category.ID {
			oldCategory = &c
			break
		}
	}

	if oldCategory == nil {
		return fmt.Errorf("category not found")
	}

	catalog, err := s.catalogRepo.GetCatalogByID(category.CatalogID)
	if err != nil {
		return err
	}

	if !canEditCatalog(catalog.Phase, userRoles) {
		return fmt.Errorf("permission denied: cannot edit catalog in %s phase", catalog.Phase)
	}

	// Log changes if in active phase
	if catalog.Phase == "active" {
		if err := s.logCategoryChanges(catalog.ID, oldCategory, category, userID); err != nil {
			return fmt.Errorf("failed to log changes: %w", err)
		}
	}

	if err := s.catalogRepo.UpdateCategory(category); err != nil {
		return err
	}

	// Audit log
	s.auditRepo.Create(&models.AuditLog{
		UserID:   &userID,
		Action:   "update",
		Resource: "category",
		Details:  fmt.Sprintf("Updated category: %s (ID: %d) in catalog %d", category.Name, category.ID, category.CatalogID),
	})

	return nil
}

// DeleteCategory deletes a category
func (s *CatalogService) DeleteCategory(categoryID, catalogID uint, userID uint, userRoles []string) error {
	catalog, err := s.catalogRepo.GetCatalogByID(catalogID)
	if err != nil {
		return err
	}
	if catalog == nil {
		return fmt.Errorf("catalog not found")
	}

	if !canEditStructure(catalog.Phase, userRoles) {
		return fmt.Errorf("permission denied: cannot delete categories in %s phase", catalog.Phase)
	}

	if err := s.catalogRepo.DeleteCategory(categoryID); err != nil {
		return err
	}

	// Audit log
	s.auditRepo.Create(&models.AuditLog{
		UserID:   &userID,
		Action:   "delete",
		Resource: "category",
		Details:  fmt.Sprintf("Deleted category ID: %d from catalog %d", categoryID, catalogID),
	})

	return nil
}

// CreateLevel creates a new level
func (s *CatalogService) CreateLevel(level *models.Level, userID uint, userRoles []string) error {
	catalog, err := s.catalogRepo.GetCatalogByID(level.CatalogID)
	if err != nil {
		return err
	}
	if catalog == nil {
		return fmt.Errorf("catalog not found")
	}

	if !canEditStructure(catalog.Phase, userRoles) {
		return fmt.Errorf("permission denied: cannot add levels in %s phase", catalog.Phase)
	}

	if err := s.catalogRepo.CreateLevel(level); err != nil {
		return err
	}

	// Audit log
	s.auditRepo.Create(&models.AuditLog{
		UserID:   &userID,
		Action:   "create",
		Resource: "level",
		Details:  fmt.Sprintf("Created level: %s (ID: %d) in catalog %d", level.Name, level.ID, level.CatalogID),
	})

	return nil
}

// UpdateLevel updates a level
func (s *CatalogService) UpdateLevel(level *models.Level, userID uint, userRoles []string) error {
	catalog, err := s.catalogRepo.GetCatalogByID(level.CatalogID)
	if err != nil {
		return err
	}

	if !canEditCatalog(catalog.Phase, userRoles) {
		return fmt.Errorf("permission denied: cannot edit catalog in %s phase", catalog.Phase)
	}

	// In active/archived phase, structural changes are not allowed
	// We can't easily check if sort_order changed without querying, but
	// the handler should prevent sort operations in active/archived phase

	if err := s.catalogRepo.UpdateLevel(level); err != nil {
		return err
	}

	// Audit log
	s.auditRepo.Create(&models.AuditLog{
		UserID:   &userID,
		Action:   "update",
		Resource: "level",
		Details:  fmt.Sprintf("Updated level: %s (ID: %d) in catalog %d", level.Name, level.ID, level.CatalogID),
	})

	return nil
}

// DeleteLevel deletes a level
func (s *CatalogService) DeleteLevel(levelID, catalogID uint, userID uint, userRoles []string) error {
	catalog, err := s.catalogRepo.GetCatalogByID(catalogID)
	if err != nil {
		return err
	}
	if catalog == nil {
		return fmt.Errorf("catalog not found")
	}

	if !canEditStructure(catalog.Phase, userRoles) {
		return fmt.Errorf("permission denied: cannot delete levels in %s phase", catalog.Phase)
	}

	if err := s.catalogRepo.DeleteLevel(levelID); err != nil {
		return err
	}

	// Audit log
	s.auditRepo.Create(&models.AuditLog{
		UserID:   &userID,
		Action:   "delete",
		Resource: "level",
		Details:  fmt.Sprintf("Deleted level ID: %d from catalog %d", levelID, catalogID),
	})

	return nil
}

// CreatePath creates a new path
func (s *CatalogService) CreatePath(path *models.Path, userID uint, userRoles []string, catalogID uint) error {
	catalog, err := s.catalogRepo.GetCatalogByID(catalogID)
	if err != nil {
		return err
	}
	if catalog == nil {
		return fmt.Errorf("catalog not found")
	}

	if !canEditStructure(catalog.Phase, userRoles) {
		return fmt.Errorf("permission denied: cannot add paths in %s phase", catalog.Phase)
	}

	if err := s.catalogRepo.CreatePath(path); err != nil {
		return err
	}

	// Audit log
	s.auditRepo.Create(&models.AuditLog{
		UserID:   &userID,
		Action:   "create",
		Resource: "path",
		Details:  fmt.Sprintf("Created path: %s (ID: %d) in catalog %d", path.Name, path.ID, catalogID),
	})

	return nil
}

// UpdatePath updates a path
func (s *CatalogService) UpdatePath(path *models.Path, userID uint, userRoles []string, catalogID uint) error {
	catalog, err := s.catalogRepo.GetCatalogByID(catalogID)
	if err != nil {
		return err
	}

	if !canEditCatalog(catalog.Phase, userRoles) {
		return fmt.Errorf("permission denied: cannot edit catalog in %s phase", catalog.Phase)
	}

	if err := s.catalogRepo.UpdatePath(path); err != nil {
		return err
	}

	// Audit log
	s.auditRepo.Create(&models.AuditLog{
		UserID:   &userID,
		Action:   "update",
		Resource: "path",
		Details:  fmt.Sprintf("Updated path: %s (ID: %d) in catalog %d", path.Name, path.ID, catalogID),
	})

	return nil
}

// DeletePath deletes a path
func (s *CatalogService) DeletePath(pathID, catalogID uint, userID uint, userRoles []string) error {
	catalog, err := s.catalogRepo.GetCatalogByID(catalogID)
	if err != nil {
		return err
	}
	if catalog == nil {
		return fmt.Errorf("catalog not found")
	}

	if !canEditStructure(catalog.Phase, userRoles) {
		return fmt.Errorf("permission denied: cannot delete paths in %s phase", catalog.Phase)
	}

	if err := s.catalogRepo.DeletePath(pathID); err != nil {
		return err
	}

	// Audit log
	s.auditRepo.Create(&models.AuditLog{
		UserID:   &userID,
		Action:   "delete",
		Resource: "path",
		Details:  fmt.Sprintf("Deleted path ID: %d from catalog %d", pathID, catalogID),
	})

	return nil
}

// CreateOrUpdateDescription creates or updates a path-level description
func (s *CatalogService) CreateOrUpdateDescription(desc *models.PathLevelDescription, userID uint, userRoles []string, catalogID uint) error {
	catalog, err := s.catalogRepo.GetCatalogByID(catalogID)
	if err != nil {
		return err
	}
	if catalog == nil {
		return fmt.Errorf("catalog not found")
	}

	if !canEditCatalog(catalog.Phase, userRoles) {
		return fmt.Errorf("permission denied: cannot edit catalog in %s phase", catalog.Phase)
	}

	// Get old description if in active phase
	if catalog.Phase == "active" {
		oldDescs, err := s.catalogRepo.GetDescriptionsByPathID(desc.PathID)
		if err != nil {
			return err
		}

		for _, oldDesc := range oldDescs {
			if oldDesc.LevelID == desc.LevelID {
				// Log the change
				change := &models.CatalogChange{
					CatalogID:  catalogID,
					EntityType: "description",
					EntityID:   oldDesc.ID,
					FieldName:  "description",
					OldValue:   &oldDesc.Description,
					NewValue:   &desc.Description,
					ChangedBy:  &userID,
				}
				if err := s.catalogRepo.LogChange(change); err != nil {
					return fmt.Errorf("failed to log change: %w", err)
				}
				break
			}
		}
	}

	return s.catalogRepo.CreatePathLevelDescription(desc)
}

// GetChangesByCatalogID retrieves all changes for a catalog
func (s *CatalogService) GetChangesByCatalogID(catalogID uint, userRoles []string) ([]models.CatalogChange, error) {
	catalog, err := s.catalogRepo.GetCatalogByID(catalogID)
	if err != nil {
		return nil, err
	}
	if catalog == nil {
		return nil, fmt.Errorf("catalog not found")
	}

	// Only admins can view change logs
	if !contains(userRoles, "admin") {
		return nil, fmt.Errorf("permission denied: only admins can view change logs")
	}

	return s.catalogRepo.GetChangesByCatalogID(catalogID)
}

// Helper functions

func canViewCatalog(phase string, userRoles []string) bool {
	isAdmin := contains(userRoles, "admin")
	isReviewer := contains(userRoles, "reviewer")

	switch phase {
	case "draft":
		return isAdmin
	case "active":
		return true // Everyone can view active phase
	case "archived":
		return isAdmin || isReviewer // Admins and reviewers can view archived
	default:
		return false
	}
}

func canEditCatalog(phase string, userRoles []string) bool {
	isAdmin := contains(userRoles, "admin")

	switch phase {
	case "draft":
		return isAdmin
	case "active", "archived":
		return false // Nobody can edit active or archived catalogs directly (use special endpoints)
	default:
		return false
	}
}

func canEditStructure(phase string, userRoles []string) bool {
	isAdmin := contains(userRoles, "admin")

	// Structural changes (create/delete/sort levels, categories, paths) only allowed in draft
	return phase == "draft" && isAdmin
}

func (s *CatalogService) validatePhaseTransition(catalogID uint, fromPhase, toPhase string, userRoles []string) error {
	if !contains(userRoles, "admin") {
		return fmt.Errorf("permission denied: only admins can change catalog phase")
	}

	// Define allowed transitions
	allowedTransitions := map[string][]string{
		"draft":    {"active"},
		"active":   {"archived", "draft"}, // Can go back to draft if no self-assessments exist
		"archived": {},                    // No transitions from archived
	}

	allowed, ok := allowedTransitions[fromPhase]
	if !ok {
		return fmt.Errorf("invalid current phase: %s", fromPhase)
	}

	validTransition := false
	for _, validPhase := range allowed {
		if toPhase == validPhase {
			validTransition = true
			break
		}
	}

	if !validTransition {
		return fmt.Errorf("cannot transition from %s to %s phase", fromPhase, toPhase)
	}

	// Additional validation when transitioning to active
	if toPhase == "active" {
		if err := s.validateCatalogCompleteness(catalogID); err != nil {
			return fmt.Errorf("cannot transition to active phase: %w", err)
		}
	}

	// Additional validation when transitioning back to draft from active
	if fromPhase == "active" && toPhase == "draft" {
		hasSelfAssessments, err := s.selfAssessmentRepo.HasSelfAssessments(catalogID)
		if err != nil {
			return fmt.Errorf("failed to check self-assessments: %w", err)
		}
		if hasSelfAssessments {
			return fmt.Errorf("cannot transition back to draft: catalog has existing self-assessments")
		}
	}

	return nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func (s *CatalogService) validateCatalogCompleteness(catalogID uint) error {
	// Get catalog with details
	catalogDetails, err := s.catalogRepo.GetCatalogWithDetails(catalogID)
	if err != nil {
		return err
	}

	// Must have at least one category
	if len(catalogDetails.Categories) == 0 {
		return fmt.Errorf("catalog must have at least one category")
	}

	// Must have at least one level
	if len(catalogDetails.Levels) == 0 {
		return fmt.Errorf("catalog must have at least one level")
	}

	// Each category must have at least one path
	for _, category := range catalogDetails.Categories {
		if len(category.Paths) == 0 {
			return fmt.Errorf("category '%s' must have at least one path", category.Name)
		}

		// Each path should have descriptions for all levels (and they should not be empty)
		for _, path := range category.Paths {
			if len(path.Descriptions) != len(catalogDetails.Levels) {
				return fmt.Errorf("path '%s' is missing descriptions for some levels", path.Name)
			}

			// Check that all descriptions are filled
			for _, desc := range path.Descriptions {
				if desc.Description == "" {
					return fmt.Errorf("path '%s' has empty descriptions", path.Name)
				}
			}
		}
	}

	return nil
}

func (s *CatalogService) logCatalogChanges(oldCatalog, newCatalog *models.CriteriaCatalog, userID uint) error {
	changes := []models.CatalogChange{}

	if oldCatalog.Name != newCatalog.Name {
		oldVal := oldCatalog.Name
		newVal := newCatalog.Name
		changes = append(changes, models.CatalogChange{
			CatalogID:  oldCatalog.ID,
			EntityType: "catalog",
			EntityID:   oldCatalog.ID,
			FieldName:  "name",
			OldValue:   &oldVal,
			NewValue:   &newVal,
			ChangedBy:  &userID,
		})
	}

	if (oldCatalog.Description == nil && newCatalog.Description != nil) ||
		(oldCatalog.Description != nil && newCatalog.Description == nil) ||
		(oldCatalog.Description != nil && newCatalog.Description != nil && *oldCatalog.Description != *newCatalog.Description) {
		changes = append(changes, models.CatalogChange{
			CatalogID:  oldCatalog.ID,
			EntityType: "catalog",
			EntityID:   oldCatalog.ID,
			FieldName:  "description",
			OldValue:   oldCatalog.Description,
			NewValue:   newCatalog.Description,
			ChangedBy:  &userID,
		})
	}

	if !oldCatalog.ValidFrom.Equal(newCatalog.ValidFrom) {
		oldVal := oldCatalog.ValidFrom.Format(time.RFC3339)
		newVal := newCatalog.ValidFrom.Format(time.RFC3339)
		changes = append(changes, models.CatalogChange{
			CatalogID:  oldCatalog.ID,
			EntityType: "catalog",
			EntityID:   oldCatalog.ID,
			FieldName:  "valid_from",
			OldValue:   &oldVal,
			NewValue:   &newVal,
			ChangedBy:  &userID,
		})
	}

	if !oldCatalog.ValidUntil.Equal(newCatalog.ValidUntil) {
		oldVal := oldCatalog.ValidUntil.Format(time.RFC3339)
		newVal := newCatalog.ValidUntil.Format(time.RFC3339)
		changes = append(changes, models.CatalogChange{
			CatalogID:  oldCatalog.ID,
			EntityType: "catalog",
			EntityID:   oldCatalog.ID,
			FieldName:  "valid_until",
			OldValue:   &oldVal,
			NewValue:   &newVal,
			ChangedBy:  &userID,
		})
	}

	for _, change := range changes {
		if err := s.catalogRepo.LogChange(&change); err != nil {
			return err
		}
	}

	return nil
}

func (s *CatalogService) logCategoryChanges(catalogID uint, oldCategory, newCategory *models.Category, userID uint) error {
	changes := []models.CatalogChange{}

	if oldCategory.Name != newCategory.Name {
		oldVal := oldCategory.Name
		newVal := newCategory.Name
		changes = append(changes, models.CatalogChange{
			CatalogID:  catalogID,
			EntityType: "category",
			EntityID:   oldCategory.ID,
			FieldName:  "name",
			OldValue:   &oldVal,
			NewValue:   &newVal,
			ChangedBy:  &userID,
		})
	}

	if (oldCategory.Description == nil && newCategory.Description != nil) ||
		(oldCategory.Description != nil && newCategory.Description == nil) ||
		(oldCategory.Description != nil && newCategory.Description != nil && *oldCategory.Description != *newCategory.Description) {
		changes = append(changes, models.CatalogChange{
			CatalogID:  catalogID,
			EntityType: "category",
			EntityID:   oldCategory.ID,
			FieldName:  "description",
			OldValue:   oldCategory.Description,
			NewValue:   newCategory.Description,
			ChangedBy:  &userID,
		})
	}

	for _, change := range changes {
		if err := s.catalogRepo.LogChange(&change); err != nil {
			return err
		}
	}

	return nil
}

// compareStringPointers safely compares two string pointers
// Treats nil and empty string as equivalent
func compareStringPointers(a, b *string) bool {
	// Both nil or both empty
	if (a == nil || *a == "") && (b == nil || *b == "") {
		return true
	}
	// One is nil/empty, other is not
	if (a == nil || *a == "") != (b == nil || *b == "") {
		return false
	}
	// Both have values
	return *a == *b
}
