package utils

import (
	"context"

	app_error "gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type PasswordUtil interface {
	HashPassword(ctx context.Context, password string) (string, error)
	IsPasswordValid(ctx context.Context, password string, hashedPassword string) bool
}

type passwordUtil struct {
	log *log.Logger
}

func NewPassword(log *log.Logger) PasswordUtil {
	return &passwordUtil{
		log: log,
	}
}

func (p *passwordUtil) HashPassword(ctx context.Context, password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		p.log.ErrorWithID(ctx, "[Password: HashPassword] Error hashing password", zap.Error(err))
		return "", app_error.New(err, app_error.ErrCodeGeneralServerUnavailable)
	}
	return string(hashedPassword), nil
}

func (p *passwordUtil) IsPasswordValid(ctx context.Context, password string, hashedPassword string) bool {
	err := p.checkPassword(ctx, password, hashedPassword)
	if err != nil {
		p.log.ErrorWithID(ctx, "[Password: IsPasswordValid] Error checking password", zap.Error(err))
		return false
	}
	return true
}

func (p *passwordUtil) checkPassword(ctx context.Context, password string, hashedPassword string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err != nil {
		p.log.ErrorWithID(ctx, "[Password: CheckPassword] Error checking password", zap.Error(err))
		return app_error.New(err, app_error.ErrCodeAuthInvalidPassword)
	}
	return nil
}
