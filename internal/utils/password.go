package utils

import (
	"context"

	app_error "gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type Password interface {
	HashPassword(ctx context.Context, password string) (string, error)
	CheckPassword(ctx context.Context, password string, hashedPassword string) error
}

type bcryptPassword struct {
	log *log.Logger
}

func NewPassword(log *log.Logger) Password {
	return &bcryptPassword{
		log: log,
	}
}

func (p *bcryptPassword) HashPassword(ctx context.Context, password string) (string, error) {
	p.log.InfoWithID(ctx, "[Password: HashPassword] Hashing password")

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		p.log.ErrorWithID(ctx, "[Password: HashPassword] Error hashing password", zap.Error(err))
		return "", app_error.New(err, app_error.ErrCodeGeneralServerUnavailable)
	}
	return string(hashedPassword), nil
}

func (p *bcryptPassword) CheckPassword(ctx context.Context, password string, hashedPassword string) error {
	p.log.InfoWithID(ctx, "[Password: CheckPassword] Checking password")

	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err != nil {
		p.log.ErrorWithID(ctx, "[Password: CheckPassword] Error checking password", zap.Error(err))
		return app_error.New(err, app_error.ErrCodeAuthInvalidPassword)
	}
	return nil
}
