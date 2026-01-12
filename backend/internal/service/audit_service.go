package service

import (
	"new-pay/internal/models"
	"new-pay/internal/repository"
)

// AuditService handles audit logging
type AuditService struct {
	auditRepo *repository.AuditRepository
}

// NewAuditService creates a new audit service
func NewAuditService(auditRepo *repository.AuditRepository) *AuditService {
	return &AuditService{
		auditRepo: auditRepo,
	}
}

// Log creates an audit log entry, ignoring errors
// This is the recommended way to log audit events as it won't fail the main operation
func (s *AuditService) Log(userID uint, action, resource, details string) {
	_ = s.auditRepo.Create(&models.AuditLog{
		UserID:   &userID,
		Action:   action,
		Resource: resource,
		Details:  details,
	})
}

// LogWithContext creates an audit log entry with IP and user agent, ignoring errors
func (s *AuditService) LogWithContext(userID uint, action, resource, details, ipAddress, userAgent string) {
	_ = s.auditRepo.Create(&models.AuditLog{
		UserID:    &userID,
		Action:    action,
		Resource:  resource,
		Details:   details,
		IPAddress: ipAddress,
		UserAgent: userAgent,
	})
}

// LogError creates an audit log entry and returns any error
// Use this when you need to handle audit logging errors explicitly
func (s *AuditService) LogError(userID uint, action, resource, details string) error {
	return s.auditRepo.Create(&models.AuditLog{
		UserID:   &userID,
		Action:   action,
		Resource: resource,
		Details:  details,
	})
}

// LogErrorWithContext creates an audit log entry with IP and user agent and returns any error
func (s *AuditService) LogErrorWithContext(userID uint, action, resource, details, ipAddress, userAgent string) error {
	return s.auditRepo.Create(&models.AuditLog{
		UserID:    &userID,
		Action:    action,
		Resource:  resource,
		Details:   details,
		IPAddress: ipAddress,
		UserAgent: userAgent,
	})
}
