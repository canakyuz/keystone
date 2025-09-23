package sendgrid

import (
	"context"
	"fmt"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

// EmailService implements the email service using SendGrid
type EmailService struct {
	client    *sendgrid.Client
	fromEmail string
	fromName  string
}

// NewEmailService creates a new SendGrid email service
func NewEmailService(apiKey, fromEmail, fromName string) *EmailService {
	client := sendgrid.NewSendClient(apiKey)

	return &EmailService{
		client:    client,
		fromEmail: fromEmail,
		fromName:  fromName,
	}
}

// SendWelcomeEmail sends a welcome email to a new user
func (s *EmailService) SendWelcomeEmail(ctx context.Context, to, name string) error {
	from := mail.NewEmail(s.fromName, s.fromEmail)
	toEmail := mail.NewEmail(name, to)
	subject := "Welcome to NexSpaces!"

	// HTML content
	htmlContent := fmt.Sprintf(`
		<html>
		<head>
			<title>Welcome to NexSpaces</title>
		</head>
		<body>
			<div style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto;">
				<div style="background-color: #f8f9fa; padding: 20px; text-align: center;">
					<h1 style="color: #333; margin: 0;">Welcome to NexSpaces!</h1>
				</div>
				<div style="padding: 30px 20px;">
					<h2 style="color: #333;">Hi %s,</h2>
					<p style="color: #666; line-height: 1.6;">
						Welcome to NexSpaces! We're excited to have you on board. Your account has been successfully created.
					</p>
					<p style="color: #666; line-height: 1.6;">
						With NexSpaces, you can:
					</p>
					<ul style="color: #666; line-height: 1.8;">
						<li>Access our comprehensive template marketplace</li>
						<li>Manage your team and users efficiently</li>
						<li>Scale your business with our powerful tools</li>
						<li>Get 24/7 support from our expert team</li>
					</ul>
					<div style="text-align: center; margin: 30px 0;">
						<a href="https://app.nexpaces.com/dashboard"
						   style="background-color: #007bff; color: white; padding: 12px 30px; text-decoration: none; border-radius: 5px; display: inline-block;">
							Get Started
						</a>
					</div>
					<p style="color: #666; line-height: 1.6;">
						If you have any questions, feel free to reach out to our support team.
					</p>
					<p style="color: #666;">
						Best regards,<br>
						The NexSpaces Team
					</p>
				</div>
				<div style="background-color: #f8f9fa; padding: 20px; text-align: center; color: #999; font-size: 12px;">
					<p>© 2024 NexSpaces. All rights reserved.</p>
				</div>
			</div>
		</body>
		</html>
	`, name)

	// Plain text content
	plainTextContent := fmt.Sprintf(`
Hi %s,

Welcome to NexSpaces! We're excited to have you on board. Your account has been successfully created.

With NexSpaces, you can:
- Access our comprehensive template marketplace
- Manage your team and users efficiently
- Scale your business with our powerful tools
- Get 24/7 support from our expert team

Get started: https://app.nexpaces.com/dashboard

If you have any questions, feel free to reach out to our support team.

Best regards,
The NexSpaces Team

© 2024 NexSpaces. All rights reserved.
	`, name)

	message := mail.NewSingleEmail(from, subject, toEmail, plainTextContent, htmlContent)

	response, err := s.client.Send(message)
	if err != nil {
		return fmt.Errorf("failed to send welcome email: %w", err)
	}

	if response.StatusCode >= 400 {
		return fmt.Errorf("SendGrid API error: status %d, body: %s", response.StatusCode, response.Body)
	}

	return nil
}

// SendInvitationEmail sends an invitation email
func (s *EmailService) SendInvitationEmail(ctx context.Context, to, inviterName, tenantName, inviteLink string) error {
	from := mail.NewEmail(s.fromName, s.fromEmail)
	toEmail := mail.NewEmail("", to)
	subject := fmt.Sprintf("You're invited to join %s on NexSpaces", tenantName)

	// HTML content
	htmlContent := fmt.Sprintf(`
		<html>
		<head>
			<title>Invitation to Join %s</title>
		</head>
		<body>
			<div style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto;">
				<div style="background-color: #f8f9fa; padding: 20px; text-align: center;">
					<h1 style="color: #333; margin: 0;">You're Invited!</h1>
				</div>
				<div style="padding: 30px 20px;">
					<h2 style="color: #333;">Hello!</h2>
					<p style="color: #666; line-height: 1.6;">
						<strong>%s</strong> has invited you to join <strong>%s</strong> on NexSpaces.
					</p>
					<p style="color: #666; line-height: 1.6;">
						NexSpaces is a powerful platform that helps teams collaborate and manage their projects efficiently with our comprehensive template marketplace.
					</p>
					<div style="text-align: center; margin: 30px 0;">
						<a href="%s"
						   style="background-color: #28a745; color: white; padding: 12px 30px; text-decoration: none; border-radius: 5px; display: inline-block;">
							Accept Invitation
						</a>
					</div>
					<p style="color: #666; line-height: 1.6;">
						This invitation will expire in 72 hours. If you don't want to receive these emails, you can safely ignore this message.
					</p>
					<p style="color: #666;">
						Best regards,<br>
						The NexSpaces Team
					</p>
				</div>
				<div style="background-color: #f8f9fa; padding: 20px; text-align: center; color: #999; font-size: 12px;">
					<p>© 2024 NexSpaces. All rights reserved.</p>
				</div>
			</div>
		</body>
		</html>
	`, tenantName, inviterName, tenantName, inviteLink)

	// Plain text content
	plainTextContent := fmt.Sprintf(`
Hello!

%s has invited you to join %s on NexSpaces.

NexSpaces is a powerful platform that helps teams collaborate and manage their projects efficiently with our comprehensive template marketplace.

Accept your invitation: %s

This invitation will expire in 72 hours. If you don't want to receive these emails, you can safely ignore this message.

Best regards,
The NexSpaces Team

© 2024 NexSpaces. All rights reserved.
	`, inviterName, tenantName, inviteLink)

	message := mail.NewSingleEmail(from, subject, toEmail, plainTextContent, htmlContent)

	response, err := s.client.Send(message)
	if err != nil {
		return fmt.Errorf("failed to send invitation email: %w", err)
	}

	if response.StatusCode >= 400 {
		return fmt.Errorf("SendGrid API error: status %d, body: %s", response.StatusCode, response.Body)
	}

	return nil
}

// SendPasswordResetEmail sends a password reset email
func (s *EmailService) SendPasswordResetEmail(ctx context.Context, to, resetLink string) error {
	from := mail.NewEmail(s.fromName, s.fromEmail)
	toEmail := mail.NewEmail("", to)
	subject := "Reset Your NexSpaces Password"

	// HTML content
	htmlContent := fmt.Sprintf(`
		<html>
		<head>
			<title>Reset Your Password</title>
		</head>
		<body>
			<div style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto;">
				<div style="background-color: #f8f9fa; padding: 20px; text-align: center;">
					<h1 style="color: #333; margin: 0;">Password Reset Request</h1>
				</div>
				<div style="padding: 30px 20px;">
					<h2 style="color: #333;">Reset Your Password</h2>
					<p style="color: #666; line-height: 1.6;">
						We received a request to reset your password for your NexSpaces account.
					</p>
					<p style="color: #666; line-height: 1.6;">
						Click the button below to reset your password:
					</p>
					<div style="text-align: center; margin: 30px 0;">
						<a href="%s"
						   style="background-color: #dc3545; color: white; padding: 12px 30px; text-decoration: none; border-radius: 5px; display: inline-block;">
							Reset Password
						</a>
					</div>
					<p style="color: #666; line-height: 1.6;">
						<strong>Important:</strong> This link will expire in 1 hour for security reasons.
					</p>
					<p style="color: #666; line-height: 1.6;">
						If you didn't request this password reset, you can safely ignore this email. Your password will not be changed.
					</p>
					<p style="color: #666;">
						Best regards,<br>
						The NexSpaces Team
					</p>
				</div>
				<div style="background-color: #f8f9fa; padding: 20px; text-align: center; color: #999; font-size: 12px;">
					<p>© 2024 NexSpaces. All rights reserved.</p>
				</div>
			</div>
		</body>
		</html>
	`, resetLink)

	// Plain text content
	plainTextContent := fmt.Sprintf(`
Reset Your Password

We received a request to reset your password for your NexSpaces account.

Reset your password: %s

Important: This link will expire in 1 hour for security reasons.

If you didn't request this password reset, you can safely ignore this email. Your password will not be changed.

Best regards,
The NexSpaces Team

© 2024 NexSpaces. All rights reserved.
	`, resetLink)

	message := mail.NewSingleEmail(from, subject, toEmail, plainTextContent, htmlContent)

	response, err := s.client.Send(message)
	if err != nil {
		return fmt.Errorf("failed to send password reset email: %w", err)
	}

	if response.StatusCode >= 400 {
		return fmt.Errorf("SendGrid API error: status %d, body: %s", response.StatusCode, response.Body)
	}

	return nil
}

// SendSubscriptionNotification sends subscription-related notifications
func (s *EmailService) SendSubscriptionNotification(ctx context.Context, to, subject, message string) error {
	from := mail.NewEmail(s.fromName, s.fromEmail)
	toEmail := mail.NewEmail("", to)

	// HTML content
	htmlContent := fmt.Sprintf(`
		<html>
		<head>
			<title>%s</title>
		</head>
		<body>
			<div style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto;">
				<div style="background-color: #f8f9fa; padding: 20px; text-align: center;">
					<h1 style="color: #333; margin: 0;">%s</h1>
				</div>
				<div style="padding: 30px 20px;">
					<p style="color: #666; line-height: 1.6;">%s</p>
					<div style="text-align: center; margin: 30px 0;">
						<a href="https://app.nexpaces.com/billing"
						   style="background-color: #007bff; color: white; padding: 12px 30px; text-decoration: none; border-radius: 5px; display: inline-block;">
							Manage Subscription
						</a>
					</div>
					<p style="color: #666;">
						Best regards,<br>
						The NexSpaces Team
					</p>
				</div>
				<div style="background-color: #f8f9fa; padding: 20px; text-align: center; color: #999; font-size: 12px;">
					<p>© 2024 NexSpaces. All rights reserved.</p>
				</div>
			</div>
		</body>
		</html>
	`, subject, subject, message)

	// Plain text content
	plainTextContent := fmt.Sprintf(`
%s

%s

Manage your subscription: https://app.nexpaces.com/billing

Best regards,
The NexSpaces Team

© 2024 NexSpaces. All rights reserved.
	`, subject, message)

	email := mail.NewSingleEmail(from, subject, toEmail, plainTextContent, htmlContent)

	response, err := s.client.Send(email)
	if err != nil {
		return fmt.Errorf("failed to send subscription notification: %w", err)
	}

	if response.StatusCode >= 400 {
		return fmt.Errorf("SendGrid API error: status %d, body: %s", response.StatusCode, response.Body)
	}

	return nil
}
