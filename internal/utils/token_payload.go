package utils

import (
	"time"

	"github.com/google/uuid"
	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
	"gitlab.com/interview-simulation/interview-backend-server/internal/entities"
)

type TokenPayload struct {
	ID        uuid.UUID          `json:"id"`
	UserID    string             `json:"user_id"`
	Role      constants.UserRole `json:"role"`
	IssuedAt  time.Time          `json:"issued_at"`
	ExpiredAt time.Time          `json:"expires_at"`
}

func NewTokenPayload(req *entities.TokenRequest) (*TokenPayload, error) {
	tokenID, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}

	payload := &TokenPayload{
		ID:        tokenID,
		UserID:    req.UserID,
		Role:      req.Role,
		IssuedAt:  time.Now(),
		ExpiredAt: time.Now().Add(req.Duration),
	}

	return payload, nil
}

func (payload *TokenPayload) Valid() error {
	if time.Now().After(payload.ExpiredAt) {
		return constants.ErrExpiredToken
	}

	return nil
}
