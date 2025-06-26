package email

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"net/smtp"
	"time"

	// "github.com/aws/aws-sdk-go/aws"
	// "github.com/aws/aws-sdk-go/aws/credentials"
	// "github.com/aws/aws-sdk-go/aws/session"
	// "github.com/aws/aws-sdk-go/service/ses"
	"github.com/hjanuschka/go-deployd/internal/config"
)

// EmailService handles email sending via SMTP or SES
type EmailService struct {
	config *config.EmailConfig
}

// NewEmailService creates a new email service
func NewEmailService(emailConfig *config.EmailConfig) *EmailService {
	return &EmailService{
		config: emailConfig,
	}
}

// SendVerificationEmail sends an email verification message
func (e *EmailService) SendVerificationEmail(to, username, verificationToken, baseURL string) error {
	verificationURL := fmt.Sprintf("%s/_dashboard/verify?token=%s", baseURL, verificationToken)

	subject := "Verify your email address"
	htmlBody := fmt.Sprintf(`
		<html>
		<body>
			<h2>Welcome to Go-Deployd!</h2>
			<p>Hi %s,</p>
			<p>Please verify your email address by clicking the link below:</p>
			<p><a href="%s" style="background-color: #4CAF50; color: white; padding: 14px 25px; text-decoration: none; display: inline-block;">Verify Email</a></p>
			<p>Or copy and paste this URL into your browser:</p>
			<p>%s</p>
			<p>This link will expire in 24 hours.</p>
			<p>If you didn't create an account, please ignore this email.</p>
			<br>
			<p>Best regards,<br>Go-Deployd Team</p>
		</body>
		</html>
	`, username, verificationURL, verificationURL)

	textBody := fmt.Sprintf(`
Welcome to Go-Deployd!

Hi %s,

Please verify your email address by visiting this URL:
%s

This link will expire in 24 hours.

If you didn't create an account, please ignore this email.

Best regards,
Go-Deployd Team
	`, username, verificationURL)

	return e.SendEmail(to, subject, textBody, htmlBody)
}

// SendEmail sends an email using the configured provider
func (e *EmailService) SendEmail(to, subject, textBody, htmlBody string) error {
	switch e.config.Provider {
	case "ses":
		return e.sendViaSES(to, subject, textBody, htmlBody)
	case "smtp":
		return e.sendViaSMTP(to, subject, textBody, htmlBody)
	default:
		return fmt.Errorf("unsupported email provider: %s", e.config.Provider)
	}
}

// sendViaSMTP sends email via SMTP
func (e *EmailService) sendViaSMTP(to, subject, textBody, htmlBody string) error {
	if e.config.SMTP.Username == "" || e.config.SMTP.Password == "" {
		return fmt.Errorf("SMTP credentials not configured")
	}

	auth := smtp.PlainAuth("", e.config.SMTP.Username, e.config.SMTP.Password, e.config.SMTP.Host)

	// Prepare message
	headers := fmt.Sprintf("From: %s <%s>\r\n", e.config.FromName, e.config.From)
	headers += fmt.Sprintf("To: %s\r\n", to)
	headers += fmt.Sprintf("Subject: %s\r\n", subject)
	headers += "MIME-Version: 1.0\r\n"
	headers += "Content-Type: multipart/alternative; boundary=boundary\r\n\r\n"

	body := "--boundary\r\n"
	body += "Content-Type: text/plain; charset=UTF-8\r\n\r\n"
	body += textBody + "\r\n\r\n"
	body += "--boundary\r\n"
	body += "Content-Type: text/html; charset=UTF-8\r\n\r\n"
	body += htmlBody + "\r\n\r\n"
	body += "--boundary--\r\n"

	message := headers + body

	serverAddr := fmt.Sprintf("%s:%d", e.config.SMTP.Host, e.config.SMTP.Port)

	if e.config.SMTP.TLS {
		// Use TLS connection
		tlsConfig := &tls.Config{
			ServerName: e.config.SMTP.Host,
		}

		conn, err := tls.Dial("tcp", serverAddr, tlsConfig)
		if err != nil {
			return fmt.Errorf("failed to connect to SMTP server with TLS: %w", err)
		}
		defer conn.Close()

		client, err := smtp.NewClient(conn, e.config.SMTP.Host)
		if err != nil {
			return fmt.Errorf("failed to create SMTP client: %w", err)
		}
		defer client.Close()

		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP authentication failed: %w", err)
		}

		if err := client.Mail(e.config.From); err != nil {
			return fmt.Errorf("failed to set sender: %w", err)
		}

		if err := client.Rcpt(to); err != nil {
			return fmt.Errorf("failed to set recipient: %w", err)
		}

		writer, err := client.Data()
		if err != nil {
			return fmt.Errorf("failed to start data transfer: %w", err)
		}

		_, err = writer.Write([]byte(message))
		if err != nil {
			return fmt.Errorf("failed to write message: %w", err)
		}

		return writer.Close()
	} else {
		// Use plain SMTP
		return smtp.SendMail(serverAddr, auth, e.config.From, []string{to}, []byte(message))
	}
}

// sendViaSES sends email via AWS SES
func (e *EmailService) sendViaSES(to, subject, textBody, htmlBody string) error {
	// TODO: Implement SES after installing AWS SDK
	return fmt.Errorf("SES support temporarily disabled - please use SMTP provider")

	/*
		if e.config.SES.AccessKeyID == "" || e.config.SES.SecretAccessKey == "" {
			return fmt.Errorf("AWS SES credentials not configured")
		}

		sess, err := session.NewSession(&aws.Config{
			Region: aws.String(e.config.SES.Region),
			Credentials: credentials.NewStaticCredentials(
				e.config.SES.AccessKeyID,
				e.config.SES.SecretAccessKey,
				"",
			),
		})
		if err != nil {
			return fmt.Errorf("failed to create AWS session: %w", err)
		}

		svc := ses.New(sess)

		input := &ses.SendEmailInput{
			Destination: &ses.Destination{
				ToAddresses: []*string{aws.String(to)},
			},
			Message: &ses.Message{
				Body: &ses.Body{
					Html: &ses.Content{
						Charset: aws.String("UTF-8"),
						Data:    aws.String(htmlBody),
					},
					Text: &ses.Content{
						Charset: aws.String("UTF-8"),
						Data:    aws.String(textBody),
					},
				},
				Subject: &ses.Content{
					Charset: aws.String("UTF-8"),
					Data:    aws.String(subject),
				},
			},
			Source: aws.String(fmt.Sprintf("%s <%s>", e.config.FromName, e.config.From)),
		}

		_, err = svc.SendEmail(input)
		if err != nil {
			return fmt.Errorf("failed to send email via SES: %w", err)
		}

		return nil
	*/
}

// TestEmail sends a test email to verify configuration
func (e *EmailService) TestEmail(to string) error {
	subject := "Go-Deployd Email Configuration Test"
	textBody := "This is a test email from Go-Deployd to verify your email configuration is working correctly."
	htmlBody := `
		<html>
		<body>
			<h2>Go-Deployd Email Test</h2>
			<p>This is a test email to verify your email configuration is working correctly.</p>
			<p>If you received this email, your email settings are properly configured!</p>
			<p>Provider: ` + e.config.Provider + `</p>
			<p>Sent at: ` + time.Now().Format(time.RFC3339) + `</p>
		</body>
		</html>
	`

	return e.SendEmail(to, subject, textBody, htmlBody)
}

// GenerateVerificationToken generates a secure random token for email verification
func GenerateVerificationToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate verification token: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// IsConfigured checks if email service is properly configured
func (e *EmailService) IsConfigured() bool {
	switch e.config.Provider {
	case "smtp":
		return e.config.SMTP.Host != "" && e.config.SMTP.Username != "" && e.config.SMTP.Password != ""
	case "ses":
		return e.config.SES.AccessKeyID != "" && e.config.SES.SecretAccessKey != "" && e.config.SES.Region != ""
	default:
		return false
	}
}
