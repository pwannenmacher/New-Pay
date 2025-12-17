package email

import (
	"bytes"
	"fmt"
	"html/template"
	"log/slog"
	"net"
	"net/smtp"

	"new-pay/internal/config"
)

// Service handles email operations
type Service struct {
	config *config.EmailConfig
}

// NewService creates a new email service
func NewService(cfg *config.EmailConfig) *Service {
	return &Service{
		config: cfg,
	}
}

// SendVerificationEmail sends an email verification email
func (s *Service) SendVerificationEmail(to, token string) error {
	subject := "Verify Your Email - NewPay"
	verificationURL := fmt.Sprintf("%s?token=%s", s.config.VerificationURL, token)

	body := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Email Verification</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <h2 style="color: #4a90e2;">Welcome to NewPay!</h2>
        <p>Thank you for registering with NewPay. Please verify your email address by clicking the button below:</p>
        <div style="text-align: center; margin: 30px 0;">
            <a href="%s" style="background-color: #4a90e2; color: white; padding: 12px 30px; text-decoration: none; border-radius: 5px; display: inline-block;">Verify Email</a>
        </div>
        <p>If the button doesn't work, you can also copy and paste the following link into your browser:</p>
        <p style="word-break: break-all; color: #4a90e2;">%s</p>
        <p>This link will expire in 24 hours.</p>
        <p>If you didn't create an account with NewPay, please ignore this email.</p>
        <hr style="border: none; border-top: 1px solid #eee; margin: 20px 0;">
        <p style="color: #999; font-size: 12px;">This is an automated email. Please do not reply.</p>
    </div>
</body>
</html>
`, verificationURL, verificationURL)

	return s.sendEmail(to, subject, body)
}

// SendPasswordResetEmail sends a password reset email
func (s *Service) SendPasswordResetEmail(to, token string) error {
	subject := "Password Reset Request - NewPay"
	resetURL := fmt.Sprintf("%s?token=%s", s.config.PasswordResetURL, token)

	body := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Password Reset</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <h2 style="color: #4a90e2;">Password Reset Request</h2>
        <p>We received a request to reset your password for your NewPay account.</p>
        <p>Click the button below to reset your password:</p>
        <div style="text-align: center; margin: 30px 0;">
            <a href="%s" style="background-color: #4a90e2; color: white; padding: 12px 30px; text-decoration: none; border-radius: 5px; display: inline-block;">Reset Password</a>
        </div>
        <p>If the button doesn't work, you can also copy and paste the following link into your browser:</p>
        <p style="word-break: break-all; color: #4a90e2;">%s</p>
        <p>This link will expire in 1 hour.</p>
        <p>If you didn't request a password reset, please ignore this email. Your password will not be changed.</p>
        <hr style="border: none; border-top: 1px solid #eee; margin: 20px 0;">
        <p style="color: #999; font-size: 12px;">This is an automated email. Please do not reply.</p>
    </div>
</body>
</html>
`, resetURL, resetURL)

	return s.sendEmail(to, subject, body)
}

// SendWelcomeEmail sends a welcome email after successful verification
func (s *Service) SendWelcomeEmail(to, name string) error {
	subject := "Welcome to NewPay!"

	body := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Welcome to NewPay</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <h2 style="color: #4a90e2;">Welcome to NewPay, %s!</h2>
        <p>Your email has been successfully verified. You can now access all features of NewPay.</p>
        <p>Here are some things you can do:</p>
        <ul>
            <li>View salary estimates for various positions</li>
            <li>Submit peer reviews</li>
            <li>Compare compensation packages</li>
        </ul>
        <p>If you have any questions or need assistance, feel free to contact our support team.</p>
        <hr style="border: none; border-top: 1px solid #eee; margin: 20px 0;">
        <p style="color: #999; font-size: 12px;">This is an automated email. Please do not reply.</p>
    </div>
</body>
</html>
`, name)

	return s.sendEmail(to, subject, body)
}

// sendEmail sends an email using SMTP
func (s *Service) sendEmail(to, subject, body string) error {
	// Create the email message
	headers := make(map[string]string)
	headers["From"] = s.config.SMTPFrom
	headers["To"] = to
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=UTF-8"

	// Build the message
	var message bytes.Buffer
	for k, v := range headers {
		message.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	message.WriteString("\r\n")
	message.WriteString(body)

	// Connect to SMTP server
	addr := net.JoinHostPort(s.config.SMTPHost, s.config.SMTPPort)
	slog.Debug("Attempting to connect to SMTP server",
		"address", addr,
		"host", s.config.SMTPHost,
		"port", s.config.SMTPPort,
	)

	// Establish connection
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		slog.Error("Failed to connect to SMTP server",
			"address", addr,
			"error", err,
		)
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer conn.Close()

	// Create SMTP client
	client, err := smtp.NewClient(conn, s.config.SMTPHost)
	if err != nil {
		slog.Error("Failed to create SMTP client", "error", err)
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer client.Close()

	// Authenticate only if credentials are provided and not empty
	// For development (e.g., Mailpit), no authentication is needed
	if s.config.SMTPUsername != "" && s.config.SMTPPassword != "" {
		auth := smtp.PlainAuth("", s.config.SMTPUsername, s.config.SMTPPassword, s.config.SMTPHost)
		// Try to authenticate, but don't fail if it's not supported (e.g., Mailpit)
		_ = client.Auth(auth)
	}

	// Set sender
	if err := client.Mail(s.config.SMTPFrom); err != nil {
		slog.Error("Failed to set sender",
			"from", s.config.SMTPFrom,
			"error", err,
		)
		return fmt.Errorf("failed to set sender: %w", err)
	}

	// Set recipient
	if err := client.Rcpt(to); err != nil {
		slog.Error("Failed to set recipient",
			"to", to,
			"error", err,
		)
		return fmt.Errorf("failed to set recipient: %w", err)
	}

	// Send message
	wc, err := client.Data()
	if err != nil {
		slog.Error("Failed to initiate data transfer", "error", err)
		return fmt.Errorf("failed to initiate data transfer: %w", err)
	}
	defer wc.Close()

	if _, err := wc.Write(message.Bytes()); err != nil {
		slog.Error("Failed to write message", "error", err)
		return fmt.Errorf("failed to write message: %w", err)
	}

	slog.Info("Email sent successfully", "to", to)

	return nil
}

// EmailTemplate represents an email template
type EmailTemplate struct {
	Subject string
	Body    *template.Template
}

// SendCatalogValidityChangeNotification sends notification about catalog validity date change
func (s *Service) SendCatalogValidityChangeNotification(to, catalogName, oldDate, newDate string) error {
	subject := fmt.Sprintf("Wichtig: Laufzeitänderung für Katalog '%s'", catalogName)

	body := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Katalog Laufzeitänderung</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <h2 style="color: #e74c3c;">Wichtige Änderung: Katalog-Laufzeit verkürzt</h2>
        <p>Der Gültigkeitszeitraum des Katalogs <strong>%s</strong> wurde geändert.</p>
        
        <div style="background-color: #fff3cd; border-left: 4px solid #ffc107; padding: 15px; margin: 20px 0;">
            <p style="margin: 5px 0;"><strong>Bisheriges Enddatum:</strong> %s</p>
            <p style="margin: 5px 0;"><strong>Neues Enddatum:</strong> %s</p>
        </div>
        
        <p><strong>Was bedeutet das für Sie?</strong></p>
        <ul>
            <li>Ihre offene Selbsteinschätzung für diesen Katalog sollte bis zum neuen Enddatum abgeschlossen werden.</li>
            <li>Bitte prüfen Sie Ihre Einschätzung und reichen Sie sie rechtzeitig ein.</li>
            <li>Bei Fragen wenden Sie sich bitte an das Review-Team.</li>
        </ul>
        
        <div style="text-align: center; margin: 30px 0;">
            <a href="%s" style="background-color: #4a90e2; color: white; padding: 12px 30px; text-decoration: none; border-radius: 5px; display: inline-block;">Zur Selbsteinschätzung</a>
        </div>
        
        <hr style="border: none; border-top: 1px solid #eee; margin: 20px 0;">
        <p style="color: #999; font-size: 12px;">Dies ist eine automatische Benachrichtigung. Bitte antworten Sie nicht auf diese E-Mail.</p>
    </div>
</body>
</html>
	`, catalogName, oldDate, newDate, s.config.VerificationURL)

	return s.sendEmail(to, subject, body)
}

// SendDraftReminderEmail sends reminder about draft self-assessment
func (s *Service) SendDraftReminderEmail(to, userName, catalogName string, draftID uint, daysSinceCreation int) error {
	subject := "Erinnerung: Ihre offene Selbsteinschätzung"

	assessmentURL := fmt.Sprintf("%s/self-assessments/%d/edit", s.config.VerificationURL, draftID)

	body := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Erinnerung Selbsteinschätzung</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <h2 style="color: #4a90e2;">Erinnerung: Ihre Selbsteinschätzung wartet</h2>
        <p>Hallo %s,</p>
        <p>Sie haben eine Selbsteinschätzung für den Katalog <strong>%s</strong> begonnen, diese aber noch nicht eingereicht.</p>
        
        <div style="background-color: #e3f2fd; border-left: 4px solid #2196f3; padding: 15px; margin: 20px 0;">
            <p style="margin: 5px 0;"><strong>Status:</strong> Entwurf</p>
            <p style="margin: 5px 0;"><strong>Erstellt vor:</strong> %d Tagen</p>
        </div>
        
        <p>Bitte nehmen Sie sich Zeit, Ihre Selbsteinschätzung zu vervollständigen und einzureichen.</p>
        
        <div style="text-align: center; margin: 30px 0;">
            <a href="%s" style="background-color: #4a90e2; color: white; padding: 12px 30px; text-decoration: none; border-radius: 5px; display: inline-block;">Selbsteinschätzung fortsetzen</a>
        </div>
        
        <hr style="border: none; border-top: 1px solid #eee; margin: 20px 0;">
        <p style="color: #999; font-size: 12px;">Sie erhalten diese Erinnerung wöchentlich, solange die Selbsteinschätzung im Entwurfsstatus ist.</p>
    </div>
</body>
</html>
	`, userName, catalogName, daysSinceCreation, assessmentURL)

	return s.sendEmail(to, subject, body)
}

// ReviewSummaryItem represents one assessment in the review summary
type ReviewSummaryItem struct {
	ID           uint
	UserName     string
	UserEmail    string
	CatalogName  string
	Status       string
	DaysInStatus int
}

// SendReviewerDailySummary sends daily summary of pending reviews
func (s *Service) SendReviewerDailySummary(to string, items []ReviewSummaryItem) error {
	subject := fmt.Sprintf("Tägliche Übersicht: %d offene Selbsteinschätzungen", len(items))

	if len(items) == 0 {
		return nil // Don't send empty summaries
	}

	// Build items HTML
	itemsHTML := ""
	statusColors := map[string]string{
		"submitted":  "#2196f3",
		"in_review":  "#ff9800",
		"reviewed":   "#9c27b0",
		"discussion": "#4caf50",
	}
	statusLabels := map[string]string{
		"submitted":  "Eingereicht",
		"in_review":  "In Prüfung",
		"reviewed":   "Geprüft",
		"discussion": "Besprechung",
	}

	for _, item := range items {
		color := statusColors[item.Status]
		if color == "" {
			color = "#757575"
		}
		label := statusLabels[item.Status]
		if label == "" {
			label = item.Status
		}

		itemsHTML += fmt.Sprintf(`
		<tr style="border-bottom: 1px solid #eee;">
			<td style="padding: 12px 8px;">%s<br><span style="color: #999; font-size: 12px;">%s</span></td>
			<td style="padding: 12px 8px;">%s</td>
			<td style="padding: 12px 8px;">
				<span style="background-color: %s; color: white; padding: 4px 8px; border-radius: 3px; font-size: 12px;">%s</span>
			</td>
			<td style="padding: 12px 8px; text-align: center;">%d Tage</td>
			<td style="padding: 12px 8px;">
				<a href="%s/admin/self-assessments/%d" style="color: #4a90e2; text-decoration: none;">Öffnen</a>
			</td>
		</tr>
		`, item.UserName, item.UserEmail, item.CatalogName, color, label, item.DaysInStatus, s.config.VerificationURL, item.ID)
	}

	body := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Review Übersicht</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 800px; margin: 0 auto; padding: 20px;">
        <h2 style="color: #4a90e2;">Tägliche Übersicht: Offene Selbsteinschätzungen</h2>
        <p>Sie haben aktuell <strong>%d Selbsteinschätzungen</strong> in Bearbeitung:</p>
        
        <table style="width: 100%%; border-collapse: collapse; margin: 20px 0; background-color: white; box-shadow: 0 2px 4px rgba(0,0,0,0.1);">
			<thead>
				<tr style="background-color: #f5f5f5; border-bottom: 2px solid #ddd;">
					<th style="padding: 12px 8px; text-align: left;">Benutzer</th>
					<th style="padding: 12px 8px; text-align: left;">Katalog</th>
					<th style="padding: 12px 8px; text-align: left;">Status</th>
					<th style="padding: 12px 8px; text-align: center;">Wartezeit</th>
					<th style="padding: 12px 8px; text-align: left;">Aktion</th>
				</tr>
			</thead>
			<tbody>
				%s
			</tbody>
		</table>
        
        <div style="background-color: #e3f2fd; border-left: 4px solid #2196f3; padding: 15px; margin: 20px 0;">
            <p style="margin: 5px 0;"><strong>Hinweis:</strong> Selbsteinschätzungen sollten zeitnah bearbeitet werden.</p>
            <p style="margin: 5px 0;">Bitte priorisieren Sie Einträge mit langer Wartezeit.</p>
        </div>
        
        <div style="text-align: center; margin: 30px 0;">
            <a href="%s/admin/self-assessments" style="background-color: #4a90e2; color: white; padding: 12px 30px; text-decoration: none; border-radius: 5px; display: inline-block;">Zum Admin-Bereich</a>
        </div>
        
        <hr style="border: none; border-top: 1px solid #eee; margin: 20px 0;">
        <p style="color: #999; font-size: 12px;">Sie erhalten diese Übersicht täglich. Dies ist eine automatische Benachrichtigung.</p>
    </div>
</body>
</html>
	`, len(items), itemsHTML, s.config.VerificationURL)

	return s.sendEmail(to, subject, body)
}

// SendReviewCompletedNotification sends notification when all reviewers have approved the final consolidation
func (s *Service) SendReviewCompletedNotification(to, userName, catalogName string, assessmentID uint) error {
	subject := "Ihre Selbsteinschätzung wurde abgeschlossen"

	// Note: At this point, the user cannot yet view the results - this will be implemented later
	body := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Selbsteinschätzung abgeschlossen</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <h2 style="color: #27ae60;">Selbsteinschätzung abgeschlossen</h2>
        <p>Hallo %s,</p>
        <p>Ihre Selbsteinschätzung für den Katalog <strong>%s</strong> wurde vom Review-Team vollständig konsolidiert und abgeschlossen.</p>
        
        <div style="background-color: #d4edda; border-left: 4px solid #28a745; padding: 15px; margin: 20px 0;">
            <p style="margin: 5px 0;"><strong>Status:</strong> Abgeschlossen (reviewed)</p>
            <p style="margin: 5px 0;"><strong>Assessment-ID:</strong> #%d</p>
        </div>
        
        <p><strong>Nächste Schritte:</strong></p>
        <ul>
            <li>Die Ergebnisse werden Ihnen in Kürze zur Einsicht freigegeben.</li>
            <li>Sie werden eine weitere Benachrichtigung erhalten, sobald Sie die detaillierten Ergebnisse einsehen können.</li>
            <li>Bei Fragen zur Bewertung können Sie sich an das Review-Team wenden.</li>
        </ul>
        
        <p>Vielen Dank für Ihre Teilnahme am Selbsteinschätzungsprozess!</p>
        
        <hr style="border: none; border-top: 1px solid #eee; margin: 20px 0;">
        <p style="color: #999; font-size: 12px;">Dies ist eine automatische Benachrichtigung. Bitte antworten Sie nicht auf diese E-Mail.</p>
    </div>
</body>
</html>
	`, userName, catalogName, assessmentID)

	return s.sendEmail(to, subject, body)
}
