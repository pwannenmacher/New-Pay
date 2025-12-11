package scheduler

import (
	"fmt"
	"log/slog"
	"new-pay/internal/email"
	"new-pay/internal/repository"
	"time"
)

// Scheduler handles periodic tasks
type Scheduler struct {
	selfAssessmentRepo *repository.SelfAssessmentRepository
	userRepo           *repository.UserRepository
	roleRepo           *repository.RoleRepository
	emailService       *email.Service
	stopChan           chan bool
}

// NewScheduler creates a new scheduler
func NewScheduler(
	selfAssessmentRepo *repository.SelfAssessmentRepository,
	userRepo *repository.UserRepository,
	roleRepo *repository.RoleRepository,
	emailService *email.Service,
) *Scheduler {
	return &Scheduler{
		selfAssessmentRepo: selfAssessmentRepo,
		userRepo:           userRepo,
		roleRepo:           roleRepo,
		emailService:       emailService,
		stopChan:           make(chan bool),
	}
}

// Start starts all scheduled tasks
func (s *Scheduler) Start() {
	slog.Info("Starting scheduler")

	// Weekly draft reminders (every Monday at 9 AM)
	go s.scheduleWeeklyTask(time.Monday, 9, 0, s.sendDraftReminders)

	// Daily reviewer summaries (every day at 8 AM)
	go s.scheduleDailyTask(8, 0, s.sendReviewerSummaries)

	slog.Info("Scheduler started")
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	slog.Info("Stopping scheduler")
	close(s.stopChan)
}

// scheduleWeeklyTask runs a task weekly on a specific weekday and time
func (s *Scheduler) scheduleWeeklyTask(weekday time.Weekday, hour, minute int, task func()) {
	for {
		now := time.Now()
		next := s.nextWeekday(now, weekday, hour, minute)
		duration := next.Sub(now)

		slog.Info("Next weekly task scheduled", "task", "draft_reminders", "next_run", next.Format("2006-01-02 15:04:05"))

		select {
		case <-time.After(duration):
			slog.Info("Running weekly task", "task", "draft_reminders")
			task()
		case <-s.stopChan:
			return
		}
	}
}

// scheduleDailyTask runs a task daily at a specific time
func (s *Scheduler) scheduleDailyTask(hour, minute int, task func()) {
	for {
		now := time.Now()
		next := s.nextDailyRun(now, hour, minute)
		duration := next.Sub(now)

		slog.Info("Next daily task scheduled", "task", "reviewer_summaries", "next_run", next.Format("2006-01-02 15:04:05"))

		select {
		case <-time.After(duration):
			slog.Info("Running daily task", "task", "reviewer_summaries")
			task()
		case <-s.stopChan:
			return
		}
	}
}

// nextWeekday calculates the next occurrence of a specific weekday and time
func (s *Scheduler) nextWeekday(from time.Time, weekday time.Weekday, hour, minute int) time.Time {
	// Start with today at the specified time
	next := time.Date(from.Year(), from.Month(), from.Day(), hour, minute, 0, 0, from.Location())

	// Calculate days until target weekday
	daysUntil := int(weekday - from.Weekday())
	if daysUntil < 0 {
		daysUntil += 7
	}

	next = next.AddDate(0, 0, daysUntil)

	// If the calculated time has already passed today, add 7 days
	if next.Before(from) || next.Equal(from) {
		next = next.AddDate(0, 0, 7)
	}

	return next
}

// nextDailyRun calculates the next daily run time
func (s *Scheduler) nextDailyRun(from time.Time, hour, minute int) time.Time {
	next := time.Date(from.Year(), from.Month(), from.Day(), hour, minute, 0, 0, from.Location())

	// If the time has already passed today, schedule for tomorrow
	if next.Before(from) || next.Equal(from) {
		next = next.AddDate(0, 0, 1)
	}

	return next
}

// sendDraftReminders sends reminders for draft self-assessments older than 7 days
func (s *Scheduler) sendDraftReminders() {
	slog.Info("Sending draft reminders")

	// Get all draft self-assessments
	assessments, err := s.selfAssessmentRepo.GetByStatus("draft")
	if err != nil {
		slog.Error("Failed to get draft assessments", "error", err)
		return
	}

	now := time.Now()
	remindersSent := 0

	for _, assessment := range assessments {
		// Check if assessment is older than 7 days
		daysSinceCreation := int(now.Sub(assessment.CreatedAt).Hours() / 24)

		// Send reminder every 7 days (7, 14, 21, etc.)
		if daysSinceCreation > 0 && daysSinceCreation%7 == 0 {
			// Get user details
			user, err := s.userRepo.GetByID(assessment.UserID)
			if err != nil || user == nil {
				slog.Error("Failed to get user", "user_id", assessment.UserID, "error", err)
				continue
			}

			userName := fmt.Sprintf("%s %s", user.FirstName, user.LastName)

			// Send reminder email
			err = s.emailService.SendDraftReminderEmail(
				user.Email,
				userName,
				assessment.CatalogName,
				assessment.ID,
				daysSinceCreation,
			)
			if err != nil {
				slog.Error("Failed to send draft reminder",
					"assessment_id", assessment.ID,
					"user_email", user.Email,
					"error", err,
				)
				continue
			}

			remindersSent++
			slog.Info("Draft reminder sent",
				"assessment_id", assessment.ID,
				"user_email", user.Email,
				"days_old", daysSinceCreation,
			)
		}
	}

	slog.Info("Draft reminders completed", "reminders_sent", remindersSent)
}

// sendReviewerSummaries sends daily summaries to all reviewers
func (s *Scheduler) sendReviewerSummaries() {
	slog.Info("Sending reviewer summaries")

	// Get all reviewers
	reviewers, err := s.roleRepo.GetUsersByRole("reviewer")
	if err != nil {
		slog.Error("Failed to get reviewers", "error", err)
		return
	}

	// Also get admins (they can review too)
	admins, err := s.roleRepo.GetUsersByRole("admin")
	if err != nil {
		slog.Error("Failed to get admins", "error", err)
	} else {
		reviewers = append(reviewers, admins...)
	}

	if len(reviewers) == 0 {
		slog.Info("No reviewers found")
		return
	}

	// Get all assessments in review states
	reviewStatuses := []string{"submitted", "in_review", "reviewed", "discussion"}
	var allItems []email.ReviewSummaryItem

	for _, status := range reviewStatuses {
		assessments, err := s.selfAssessmentRepo.GetByStatus(status)
		if err != nil {
			slog.Error("Failed to get assessments", "status", status, "error", err)
			continue
		}

		now := time.Now()
		for _, assessment := range assessments {
			// Calculate days in current status
			var statusDate time.Time
			switch status {
			case "submitted":
				if assessment.SubmittedAt != nil {
					statusDate = *assessment.SubmittedAt
				}
			case "in_review":
				if assessment.InReviewAt != nil {
					statusDate = *assessment.InReviewAt
				}
			case "reviewed":
				if assessment.ReviewedAt != nil {
					statusDate = *assessment.ReviewedAt
				}
			case "discussion":
				if assessment.DiscussionStartedAt != nil {
					statusDate = *assessment.DiscussionStartedAt
				}
			}

			daysInStatus := 0
			if !statusDate.IsZero() {
				daysInStatus = int(now.Sub(statusDate).Hours() / 24)
			}

			allItems = append(allItems, email.ReviewSummaryItem{
				ID:           assessment.ID,
				UserName:     assessment.UserName,
				UserEmail:    assessment.UserEmail,
				CatalogName:  assessment.CatalogName,
				Status:       assessment.Status,
				DaysInStatus: daysInStatus,
			})
		}
	}

	// Send summary to each reviewer
	summariesSent := 0
	for _, reviewer := range reviewers {
		if len(allItems) == 0 {
			continue // Don't send empty summaries
		}

		err := s.emailService.SendReviewerDailySummary(reviewer.Email, allItems)
		if err != nil {
			slog.Error("Failed to send reviewer summary",
				"reviewer_email", reviewer.Email,
				"error", err,
			)
			continue
		}

		summariesSent++
		slog.Info("Reviewer summary sent",
			"reviewer_email", reviewer.Email,
			"items_count", len(allItems),
		)
	}

	slog.Info("Reviewer summaries completed",
		"summaries_sent", summariesSent,
		"total_items", len(allItems),
	)
}
