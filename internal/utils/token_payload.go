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

type WebSocketSessionPayload struct {
	UserID     string    `json:"user_id"`
	SessionID  string    `json:"session_id"`
	ResumeID   string    `json:"resume_id"`
	Position   string    `json:"position"`
	BiasPrompt string    `json:"bias_prompt"`
	IssuedAt   time.Time `json:"issued_at"`
	ExpiredAt  time.Time `json:"expires_at"`
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

func NewWebSocketSessionPayload(req *entities.WebSocketSessionReq) *WebSocketSessionPayload {
	return &WebSocketSessionPayload{
		UserID:     req.UserID,
		SessionID:  req.SessionID,
		ResumeID:   req.ResumeID,
		Position:   req.Position,
		BiasPrompt: req.BiasPrompt,
		IssuedAt:   time.Now(),
		ExpiredAt:  time.Now().Add(req.Duration),
	}
}

func (payload *SignInTokenPayload) Valid() error {
	if time.Now().After(payload.ExpiredAt) {
		return constants.ErrExpiredToken
	}

	return nil
}

func (payload *SignInTokenPayload) ValidWithGraceWindow() error {
	graceWindow := constants.TokenGraceWindow
	if graceWindow == 0 {
		graceWindow = time.Minute
	}

	if time.Since(payload.ExpiredAt) > graceWindow {
		return constants.ErrExpiredToken
	}

	return nil
}

func (payload *SignInTokenPayload) GetExpiredAt() time.Time {
	return payload.ExpiredAt
}

func (payload *VerifyEmailTokenPayload) Valid() error {
	if time.Now().After(payload.ExpiredAt) {
		return constants.ErrExpiredToken
	}

	return nil
}

func (payload *VerifyEmailTokenPayload) ValidWithGraceWindow() error {
	graceWindow := constants.TokenGraceWindow
	if graceWindow == 0 {
		graceWindow = time.Minute
	}

	if time.Since(payload.ExpiredAt) > graceWindow {
		return constants.ErrExpiredToken
	}

	return nil
}

func (payload *VerifyEmailTokenPayload) GetExpiredAt() time.Time {
	return payload.ExpiredAt
}

func (payload *WebSocketSessionPayload) Valid() error {
	if time.Now().After(payload.ExpiredAt) {
		return constants.ErrExpiredToken
	}

	return nil
}

func (payload *WebSocketSessionPayload) ValidWithGraceWindow() error {
	graceWindow := constants.TokenGraceWindow
	if graceWindow == 0 {
		graceWindow = time.Minute
	}

	if time.Since(payload.ExpiredAt) > graceWindow {
		return constants.ErrExpiredToken
	}

	return nil
}

func (payload *WebSocketSessionPayload) GetExpiredAt() time.Time {
	return payload.ExpiredAt
}
