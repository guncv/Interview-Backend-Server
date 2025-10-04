package email

import (
	"context"
	"fmt"

	"gitlab.com/interview-simulation/interview-backend-server/internal/config"
	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
	"gopkg.in/gomail.v2"
)

type EmailSender interface {
	SendResetPasswordEmail(ctx context.Context, payload *ResetPasswordEmailPayload) error
	SendVerifyEmail(ctx context.Context, payload *VerifyEmailPayload) error
}

type emailSender struct {
	config *config.Config
	log    *log.Logger
}

type ResetPasswordEmailPayload struct {
	EmailReceiver string
	Token         string
}

type VerifyEmailPayload struct {
	EmailReceiver string
	Code          string
}

func NewEmailSender(config *config.Config, log *log.Logger) EmailSender {
	return &emailSender{
		config: config,
		log:    log,
	}
}

func (s *emailSender) SendResetPasswordEmail(ctx context.Context, payload *ResetPasswordEmailPayload) error {

	resetLink := fmt.Sprintf("%s?token=%s", s.config.EmailConfig.ResetPasswordURL, payload.Token)
	body := s.buildResetPasswordEmailBody(resetLink)

	if err := s.sendEmail(ctx, payload.EmailReceiver, constants.SubjectResetPassword, body); err != nil {
		s.log.ErrorWithID(ctx, "[Email: SendResetPasswordEmail] Failed to send email", err)
		return app_error.New(fmt.Errorf("failed to send reset password email: %w", err), app_error.ErrCodeGeneralServerUnavailable)
	}

	return nil
}

func (s *emailSender) SendVerifyEmail(ctx context.Context, payload *VerifyEmailPayload) error {

	body := s.buildVerifyEmailBody(payload.Code)

	if err := s.sendEmail(ctx, payload.EmailReceiver, constants.SubjectVerifyEmail, body); err != nil {
		s.log.ErrorWithID(ctx, "[Email: SendVerifyEmail] Failed to send email", err)
		return app_error.New(fmt.Errorf("failed to send verify email: %w", err), app_error.ErrCodeGeneralServerUnavailable)
	}

	return nil
}

func (s *emailSender) sendEmail(ctx context.Context, to, subject, body string) error {
	msg := gomail.NewMessage()
	msg.SetHeader("From", s.config.EmailConfig.From)
	msg.SetHeader("To", to)
	msg.SetHeader("Subject", subject)
	msg.SetBody("text/html", body)

	dialer := gomail.NewDialer(
		s.config.EmailConfig.Host,
		s.config.EmailConfig.Port,
		s.config.EmailConfig.Username,
		s.config.EmailConfig.Password,
	)

	if err := dialer.DialAndSend(msg); err != nil {
		s.log.ErrorWithID(ctx, "[Email: SendEmail] Failed to send email", err)
		return fmt.Errorf("failed to send email via SMTP: %w", err)
	}

	return nil
}

func (s *emailSender) buildResetPasswordEmailBody(resetLink string) string {
	return fmt.Sprintf(`
		<h2>Reset your password</h2>
		<p>Click the link below to reset your password. The link will expire in %s.</p>
		<p><a href="%s">Reset Password</a></p>
		<p>If you didn't request this, you can safely ignore this email.</p>
	`, s.config.AuthConfig.ResetPasswordTokenDuration.String(), resetLink)
}

func (s *emailSender) buildVerifyEmailBody(code string) string {
	return fmt.Sprintf(`
		<h2>Verify your email</h2>
		<p>Use the following verification code to verify your email. The code will expire in %s.</p>
		<p><strong>Verification Code: %s</strong></p>
		<p>If you didn't request this, you can safely ignore this email.</p>
	`, s.config.EmailConfig.VerifyEmailTokenDuration.String(), code)
}
