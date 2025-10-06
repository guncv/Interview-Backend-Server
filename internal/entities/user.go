package entities

import (
	"time"

	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
)

type HealthCheckResponse struct {
	Status string `json:"status"`
}

type HandleGoogleCallbackReq struct {
	Code  string `json:"code"`
	State string `json:"state"`
}

type HandleGoogleCallbackResp struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type GoogleAuthURLResponse struct {
	AuthURL string `json:"auth_url"`
}

type HandleFacebookCallbackReq struct {
	Code  string `json:"code"`
	State string `json:"state"`
}

type HandleFacebookCallbackResp struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type FacebookAuthURLResponse struct {
	AuthURL string `json:"auth_url"`
}

type TokenRequest struct {
	UserID   string             `json:"user_id"`
	Role     constants.UserRole `json:"role"`
	Duration time.Duration      `json:"duration"`
}

type VerifyEmailTokenRequest struct {
	UserID   string        `json:"user_id"`
	Email    string        `json:"email"`
	Duration time.Duration `json:"duration"`
}

type CookieRequest struct {
	RefreshToken string        `json:"refresh_token"`
	Duration     time.Duration `json:"duration"`
	Domain       string        `json:"domain"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type RefreshTokenResponse struct {
	AccessToken string `json:"access_token"`
}
