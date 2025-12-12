package service

import (
	"database/sql"
	"fmt"

	"new-pay/internal/keymanager"
	"new-pay/internal/models"
	"new-pay/internal/repository"
	"new-pay/internal/securestore"
)

// EncryptedResponseService handles encryption/decryption of assessment response justifications
type EncryptedResponseService struct {
	db           *sql.DB
	responseRepo *repository.AssessmentResponseRepository
	keyManager   *keymanager.KeyManager
	secureStore  *securestore.SecureStore
}

// NewEncryptedResponseService creates a new encrypted response service
func NewEncryptedResponseService(
	db *sql.DB,
	responseRepo *repository.AssessmentResponseRepository,
	keyManager *keymanager.KeyManager,
	secureStore *securestore.SecureStore,
) *EncryptedResponseService {
	return &EncryptedResponseService{
		db:           db,
		responseRepo: responseRepo,
		keyManager:   keyManager,
		secureStore:  secureStore,
	}
}

// CreateResponse creates a new assessment response with encrypted justification
func (s *EncryptedResponseService) CreateResponse(response *models.AssessmentResponse, userID uint) error {
	// Ensure user key exists
	if err := s.ensureUserKey(int64(userID)); err != nil {
		return fmt.Errorf("failed to ensure user key: %w", err)
	}

	// Ensure process key exists for this assessment
	processID := fmt.Sprintf("assessment-%d", response.AssessmentID)
	if err := s.ensureProcessKey(processID); err != nil {
		return fmt.Errorf("failed to ensure process key: %w", err)
	}

	// Encrypt justification
	data := &securestore.PlainData{
		Fields: map[string]interface{}{
			"justification": response.Justification,
		},
		Metadata: map[string]string{
			"assessment_id": fmt.Sprintf("%d", response.AssessmentID),
			"category_id":   fmt.Sprintf("%d", response.CategoryID),
		},
	}

	record, err := s.secureStore.CreateRecord(
		processID,
		int64(userID),
		"JUSTIFICATION",
		data,
		"",
	)
	if err != nil {
		return fmt.Errorf("failed to encrypt justification: %w", err)
	}

	// Store with encrypted_justification_id
	response.EncryptedJustificationID = &record.ID
	response.Justification = "" // Clear plaintext

	query := `
		INSERT INTO assessment_responses (assessment_id, category_id, path_id, level_id, encrypted_justification_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`
	err = s.db.QueryRow(
		query,
		response.AssessmentID,
		response.CategoryID,
		response.PathID,
		response.LevelID,
		response.EncryptedJustificationID,
	).Scan(&response.ID, &response.CreatedAt, &response.UpdatedAt)

	return err
}

// UpdateResponse updates an existing assessment response with encrypted justification
func (s *EncryptedResponseService) UpdateResponse(response *models.AssessmentResponse, userID uint) error {
	// Ensure user key exists
	if err := s.ensureUserKey(int64(userID)); err != nil {
		return fmt.Errorf("failed to ensure user key: %w", err)
	}

	// Ensure process key exists for this assessment
	processID := fmt.Sprintf("assessment-%d", response.AssessmentID)
	if err := s.ensureProcessKey(processID); err != nil {
		return fmt.Errorf("failed to ensure process key: %w", err)
	}

	// Encrypt new justification
	data := &securestore.PlainData{
		Fields: map[string]interface{}{
			"justification": response.Justification,
		},
		Metadata: map[string]string{
			"assessment_id": fmt.Sprintf("%d", response.AssessmentID),
			"category_id":   fmt.Sprintf("%d", response.CategoryID),
			"update":        "true",
		},
	}

	record, err := s.secureStore.CreateRecord(
		processID,
		int64(userID),
		"JUSTIFICATION",
		data,
		"",
	)
	if err != nil {
		return fmt.Errorf("failed to encrypt justification: %w", err)
	}

	// Update with new encrypted_justification_id (creates new encrypted record, old one remains in chain)
	response.EncryptedJustificationID = &record.ID
	response.Justification = "" // Clear plaintext

	query := `
		UPDATE assessment_responses
		SET path_id = $1, level_id = $2, encrypted_justification_id = $3, updated_at = CURRENT_TIMESTAMP
		WHERE id = $4
		RETURNING updated_at
	`
	err = s.db.QueryRow(
		query,
		response.PathID,
		response.LevelID,
		response.EncryptedJustificationID,
		response.ID,
	).Scan(&response.UpdatedAt)

	return err
}

// DecryptResponse decrypts the justification field of an assessment response
func (s *EncryptedResponseService) DecryptResponse(response *models.AssessmentResponse) error {
	if response.EncryptedJustificationID == nil {
		// Fallback: check if justification is stored in plaintext (old data)
		if response.Justification != "" {
			return nil // Already has plaintext justification
		}
		return fmt.Errorf("no justification found (neither encrypted nor plaintext)")
	}

	// Decrypt the justification
	data, err := s.secureStore.DecryptRecord(*response.EncryptedJustificationID)
	if err != nil {
		return fmt.Errorf("failed to decrypt justification: %w", err)
	}

	// Extract justification from decrypted data
	if justification, ok := data.Fields["justification"].(string); ok {
		response.Justification = justification
	} else {
		return fmt.Errorf("invalid justification data format")
	}

	return nil
}

// GetResponseByID retrieves and decrypts an assessment response
func (s *EncryptedResponseService) GetResponseByID(responseID uint) (*models.AssessmentResponse, error) {
	response, err := s.responseRepo.GetByID(responseID)
	if err != nil || response == nil {
		return response, err
	}

	// Decrypt justification
	if err := s.DecryptResponse(response); err != nil {
		return nil, err
	}

	return response, nil
}

// GetResponsesByAssessment retrieves and decrypts all responses for an assessment
func (s *EncryptedResponseService) GetResponsesByAssessment(assessmentID uint) ([]*models.AssessmentResponse, error) {
	responses, err := s.responseRepo.GetByAssessmentID(assessmentID)
	if err != nil {
		return nil, err
	}

	// Decrypt all justifications
	for _, response := range responses {
		if err := s.DecryptResponse(response); err != nil {
			return nil, fmt.Errorf("failed to decrypt response %d: %w", response.ID, err)
		}
	}

	return responses, nil
}

// GetResponsesWithDetailsByAssessment retrieves and decrypts all responses with category/path/level details
func (s *EncryptedResponseService) GetResponsesWithDetailsByAssessment(assessmentID uint) ([]*models.AssessmentResponseWithDetails, error) {
	responses, err := s.responseRepo.GetWithDetailsByAssessmentID(assessmentID)
	if err != nil {
		return nil, err
	}

	// Decrypt all justifications
	for _, response := range responses {
		if err := s.DecryptResponse(&response.AssessmentResponse); err != nil {
			return nil, fmt.Errorf("failed to decrypt response %d: %w", response.ID, err)
		}
	}

	return responses, nil
}

// ensureUserKey ensures a user has an encryption key pair
func (s *EncryptedResponseService) ensureUserKey(userID int64) error {
	// Check if user key already exists
	_, err := s.keyManager.GetUserPublicKey(userID)
	if err == nil {
		return nil // Key exists
	}

	// Create new user key
	_, err = s.keyManager.CreateUserKey(userID)
	return err
}

// ensureProcessKey ensures a process has an encryption key
func (s *EncryptedResponseService) ensureProcessKey(processID string) error {
	// Check if process key already exists
	_, err := s.keyManager.GetProcessKeyHash(processID)
	if err == nil {
		return nil // Key exists
	}

	// Create new process key (no expiration)
	return s.keyManager.CreateProcessKey(processID, nil)
}
