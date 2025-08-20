package utils

import (
	"time"

	"github.com/google/uuid"
	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
	"gitlab.com/interview-simulation/interview-backend-server/internal/entities"
)

type SignInTokenPayload struct {
	ID        uuid.UUID          `json:"id"`
	UserID    string             `json:"user_id"`
	Role      constants.UserRole `json:"role"`
	IssuedAt  time.Time          `json:"issued_at"`
	ExpiredAt time.Time          `json:"expires_at"`
}

type VerifyEmailTokenPayload struct {
	ID        uuid.UUID `json:"id"`
	UserID    string    `json:"user_id"`
	Email     string    `json:"email"`
	IssuedAt  time.Time `json:"issued_at"`
	ExpiredAt time.Time `json:"expires_at"`
}

type InterviewSessionTokenPayload struct {
	ID        uuid.UUID `json:"id"`
	SessionID uuid.UUID `json:"session_id"`
	UserID    string    `json:"user_id"`
	IssuedAt  time.Time `json:"issued_at"`
	ExpiredAt time.Time `json:"expires_at"`
}

func NewSignInTokenPayload(req *entities.TokenRequest) (*SignInTokenPayload, error) {
	tokenID, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}

	payload := &SignInTokenPayload{
		ID:        tokenID,
		UserID:    req.UserID,
		Role:      req.Role,
		IssuedAt:  time.Now(),
		ExpiredAt: time.Now().Add(req.Duration),
	}

	return payload, nil
}

func NewVerifyEmailTokenPayload(req *entities.VerifyEmailTokenRequest) (*VerifyEmailTokenPayload, error) {
	tokenID, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}

	payload := &VerifyEmailTokenPayload{
		ID:        tokenID,
		UserID:    req.UserID,
		Email:     req.Email,
		IssuedAt:  time.Now(),
		ExpiredAt: time.Now().Add(req.Duration),
	}

	return payload, nil
}

func NewInterviewSessionTokenPayload(req *entities.CreateInterviewSessionTokenReq) (*InterviewSessionTokenPayload, error) {
	tokenID, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}

	payload := &InterviewSessionTokenPayload{
		ID:        tokenID,
		SessionID: req.SessionID,
		UserID:    req.UserID,
		IssuedAt:  time.Now(),
		ExpiredAt: time.Now().Add(req.Duration),
	}

	return payload, nil
}

func (payload *InterviewSessionTokenPayload) Valid() error {
	if time.Now().After(payload.ExpiredAt) {
		return constants.ErrExpiredToken
	}

	return nil
}

func (payload *SignInTokenPayload) Valid() error {
	if time.Now().After(payload.ExpiredAt) {
		return constants.ErrExpiredToken
	}

	return nil
}

func (payload *VerifyEmailTokenPayload) Valid() error {
	if time.Now().After(payload.ExpiredAt) {
		return constants.ErrExpiredToken
	}

	return nil
}
