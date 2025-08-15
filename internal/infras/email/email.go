package email

import (
	"context"
	"fmt"

	"gitlab.com/interview-simulation/interview-backend-server/internal/config"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
	"gopkg.in/gomail.v2"
)

type EmailSender interface {
	SendResetPasswordEmail(ctx context.Context, payload *ResetPasswordEmailPayload) error
}

type emailSender struct {
	config *config.Config
	log    *log.Logger
}

type ResetPasswordEmailPayload struct {
	EmailReceiver string
	Token         string
}

func NewEmailSender(config *config.Config, log *log.Logger) EmailSender {
	return &emailSender{
		config: config,
		log:    log,
	}
}

func (s *emailSender) SendResetPasswordEmail(ctx context.Context, payload *ResetPasswordEmailPayload) error {
	s.log.InfoWithID(ctx, "[Email: SendResetPasswordEmail] Called")
	subject := "Reset your password"

	resetLink := fmt.Sprintf("%s?token=%s", s.config.EmailConfig.ResetPasswordURL, payload.Token)
	body := fmt.Sprintf(`
		<h2>Reset your password</h2>
		<p>Click the link below to reset your password. The link will expire in 15 minutes.</p>
		<p><a href="%s">%s</a></p>
		<p>If you didn’t request this, you can safely ignore this email.</p>
	`, resetLink, resetLink)

	msg := gomail.NewMessage()
	msg.SetHeader("From", s.config.EmailConfig.From)
	msg.SetHeader("To", payload.EmailReceiver)
	msg.SetHeader("Subject", subject)
	msg.SetBody("text/html", body)

	d := gomail.NewDialer(s.config.EmailConfig.Host, s.config.EmailConfig.Port, s.config.EmailConfig.Username, s.config.EmailConfig.Password)
	if err := d.DialAndSend(msg); err != nil {
		s.log.ErrorWithID(ctx, "[Email: SendResetPasswordEmail] Error sending email", err)
		return app_error.New(err, app_error.ErrCodeGeneralServerUnavailable)
	}

	return nil
}
