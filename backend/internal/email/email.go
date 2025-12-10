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
