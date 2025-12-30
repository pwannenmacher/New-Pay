package scheduler

import (
	"fmt"
	"log/slog"
	"new-pay/internal/config"
	"new-pay/internal/email"
	"new-pay/internal/models"
	"new-pay/internal/repository"
	"strconv"
	"strings"
	"time"
)

// Scheduler handles periodic tasks
type Scheduler struct {
	selfAssessmentRepo *repository.SelfAssessmentRepository
	userRepo           *repository.UserRepository
	roleRepo           *repository.RoleRepository
	emailService       *email.Service
	config             *config.SchedulerConfig
	stopChan           chan bool
}

// NewScheduler creates a new scheduler
func NewScheduler(
	selfAssessmentRepo *repository.SelfAssessmentRepository,
	userRepo *repository.UserRepository,
	roleRepo *repository.RoleRepository,
	emailService *email.Service,
	cfg *config.SchedulerConfig,
) *Scheduler {
	return &Scheduler{
		selfAssessmentRepo: selfAssessmentRepo,
		userRepo:           userRepo,
		roleRepo:           roleRepo,
		emailService:       emailService,
		config:             cfg,
		stopChan:           make(chan bool),
	}
}

// Start starts all scheduled tasks
func (s *Scheduler) Start() {
	slog.Info("Starting scheduler", "draft_reminders_enabled", s.config.EnableDraftReminders, "reviewer_summary_enabled", s.config.EnableReviewerSummary)

	if s.config.EnableDraftReminders {
		// Parse cron and start draft reminders
		if err := s.startCronTask(s.config.DraftReminderCron, "draft_reminders", s.sendDraftReminders); err != nil {
			slog.Error("Failed to start draft reminders", "error", err)
		}
	}

	if s.config.EnableReviewerSummary {
		// Parse cron and start reviewer summaries
		if err := s.startCronTask(s.config.ReviewerSummaryCron, "reviewer_summaries", s.sendReviewerSummaries); err != nil {
			slog.Error("Failed to start reviewer summaries", "error", err)
		}
	}

	slog.Info("Scheduler started")
}

// startCronTask parses a cron expression and starts the task
// Supports simple cron format: "minute hour day month weekday"
// Examples: "0 9 * * 1" = Monday 9 AM, "0 8 * * *" = Daily 8 AM, "*/5 * * * *" = Every 5 minutes
func (s *Scheduler) startCronTask(cronExpr, taskName string, task func()) error {
	parts := strings.Fields(cronExpr)
	if len(parts) != 5 {
		return fmt.Errorf("invalid cron expression: %s (expected 5 fields)", cronExpr)
	}

	// Parse minute field (supports */n for intervals)
	if strings.HasPrefix(parts[0], "*/") {
		// Interval notation: */5 = every 5 minutes
		interval, err := strconv.Atoi(parts[0][2:])
		if err != nil || interval < 1 || interval > 59 {
			return fmt.Errorf("invalid minute interval in cron: %s", parts[0])
		}
		// For interval tasks, run immediately
		go s.scheduleIntervalTask(time.Duration(interval)*time.Minute, taskName, task)
		return nil
	}

	minute, err := strconv.Atoi(parts[0])
	if err != nil || minute < 0 || minute > 59 {
		return fmt.Errorf("invalid minute in cron: %s", parts[0])
	}

	// Parse hour field (supports */n for intervals)
	if strings.HasPrefix(parts[1], "*/") {
		// Interval notation: */2 = every 2 hours
		interval, err := strconv.Atoi(parts[1][2:])
		if err != nil || interval < 1 || interval > 23 {
			return fmt.Errorf("invalid hour interval in cron: %s", parts[1])
		}
		// For hourly intervals at a specific minute
		go s.scheduleHourlyIntervalTask(interval, minute, taskName, task)
		return nil
	}

	hour, err := strconv.Atoi(parts[1])
	if err != nil || hour < 0 || hour > 23 {
		return fmt.Errorf("invalid hour in cron: %s", parts[1])
	}

	// Check if daily or weekly
	if parts[4] == "*" {
		// Daily task
		go s.scheduleDailyTask(hour, minute, taskName, task)
	} else {
		// Weekly task
		weekday, err := strconv.Atoi(parts[4])
		if err != nil || weekday < 0 || weekday > 6 {
			return fmt.Errorf("invalid weekday in cron: %s (0-6, 0=Sunday)", parts[4])
		}
		go s.scheduleWeeklyTask(time.Weekday(weekday), hour, minute, taskName, task)
	}

	return nil
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	slog.Info("Stopping scheduler")
	close(s.stopChan)
}

// scheduleIntervalTask runs a task at regular intervals
func (s *Scheduler) scheduleIntervalTask(interval time.Duration, taskName string, task func()) {
	slog.Info("Starting interval task", "task", taskName, "interval", interval)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Run immediately on start
	slog.Info("Running interval task", "task", taskName)
	task()

	for {
		select {
		case <-ticker.C:
			slog.Info("Running interval task", "task", taskName)
			task()
		case <-s.stopChan:
			return
		}
	}
}

// scheduleHourlyIntervalTask runs a task every N hours at a specific minute
func (s *Scheduler) scheduleHourlyIntervalTask(hourInterval, minute int, taskName string, task func()) {
	slog.Info("Starting hourly interval task", "task", taskName, "interval_hours", hourInterval, "minute", minute)

	for {
		now := time.Now()
		next := s.nextHourlyInterval(now, hourInterval, minute)
		duration := next.Sub(now)

		slog.Info("Next hourly interval task scheduled", "task", taskName, "next_run", next.Format("2006-01-02 15:04:05"))

		select {
		case <-time.After(duration):
			slog.Info("Running hourly interval task", "task", taskName)
			task()
		case <-s.stopChan:
			return
		}
	}
}

// nextHourlyInterval calculates the next run time for hourly intervals
func (s *Scheduler) nextHourlyInterval(from time.Time, hourInterval, minute int) time.Time {
	// Start with current hour at the specified minute
	next := time.Date(from.Year(), from.Month(), from.Day(), from.Hour(), minute, 0, 0, from.Location())

	// If the time has passed in this hour, move to next hour
	if next.Before(from) || next.Equal(from) {
		next = next.Add(time.Hour)
	}

	// Find the next hour that matches the interval
	for next.Hour()%hourInterval != 0 {
		next = next.Add(time.Hour)
	}

	return next
}

// scheduleWeeklyTask runs a task weekly on a specific weekday and time
func (s *Scheduler) scheduleWeeklyTask(weekday time.Weekday, hour, minute int, taskName string, task func()) {
	for {
		now := time.Now()
		next := s.nextWeekday(now, weekday, hour, minute)
		duration := next.Sub(now)

		slog.Info("Next weekly task scheduled", "task", taskName, "next_run", next.Format("2006-01-02 15:04:05"))

		select {
		case <-time.After(duration):
			slog.Info("Running weekly task", "task", taskName)
			task()
		case <-s.stopChan:
			return
		}
	}
}

// scheduleDailyTask runs a task daily at a specific time
func (s *Scheduler) scheduleDailyTask(hour, minute int, taskName string, task func()) {
	for {
		now := time.Now()
		next := s.nextDailyRun(now, hour, minute)
		duration := next.Sub(now)

		slog.Info("Next daily task scheduled", "task", taskName, "next_run", next.Format("2006-01-02 15:04:05"))

		select {
		case <-time.After(duration):
			slog.Info("Running daily task", "task", taskName)
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
	reminderIntervalMins := s.config.ReminderIntervalMins

	for _, assessment := range assessments {
		// Calculate minutes since creation
		minutesSinceCreation := int(now.Sub(assessment.CreatedAt).Minutes())

		// Send reminder at each interval (e.g., 10080 mins = 7 days, 20160 = 14 days, etc.)
		if minutesSinceCreation > 0 && minutesSinceCreation%reminderIntervalMins == 0 {
			// Get user details
			user, err := s.userRepo.GetByID(assessment.UserID)
			if err != nil || user == nil {
				slog.Error("Failed to get user", "user_id", assessment.UserID, "error", err)
				continue
			}

			userName := fmt.Sprintf("%s %s", user.FirstName, user.LastName)
			daysSinceCreation := int(now.Sub(assessment.CreatedAt).Hours() / 24)

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

	// Deduplicate by user ID (ignore admin role if user is already a reviewer)
	uniqueReviewers := make(map[uint]models.User)
	for _, reviewer := range reviewers {
		uniqueReviewers[reviewer.ID] = reviewer
	}

	// Convert map back to slice
	reviewers = make([]models.User, 0, len(uniqueReviewers))
	for _, reviewer := range uniqueReviewers {
		reviewers = append(reviewers, reviewer)
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
