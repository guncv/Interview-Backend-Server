package entities

import (
	"time"

	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
)

type HealthCheckResponse struct {
	Status string `json:"status"`
}

type SignUpUserRequest struct {
	Email       string    `json:"email" validate:"required,email,max=100"`
	Password    string    `json:"password" validate:"required,min=8,max=100"`
	FullName    string    `json:"full_name" validate:"required,min=2,max=50"`
	PhoneNumber string    `json:"phone_number" validate:"required,min=10,max=20"`
	Country     string    `json:"country" validate:"required,min=2,max=50"`
	City        string    `json:"city" validate:"required,min=2,max=50"`
	Address     string    `json:"address" validate:"required,min=5,max=200"`
	PostalCode  string    `json:"postal_code" validate:"required,min=5,max=20"`
	Gender      string    `json:"gender" validate:"required,oneof=male female other"`
	DateOfBirth time.Time `json:"date_of_birth" validate:"required"`
}

type SignUpUserResponse struct {
	TokenId string `json:"token_id"`
}

type VerifyEmailRequest struct {
	Token string `json:"token" validate:"required"`
	Code  string `json:"code" validate:"required,min=6,max=6"`
}

type ResetVerifyEmailCodeRequest struct {
	Token string `json:"token" validate:"required"`
}

type ResetVerifyEmailCodeResponse struct {
	TokenId string `json:"token_id"`
}

type SignInUserByEmailAndPasswordRequest struct {
	Email    string `json:"email" validate:"required,email,max=100"`
	Password string `json:"password" validate:"required,min=8,max=100"`
}

type SignInUserByEmailAndPasswordResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
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

type ForgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email,max=100"`
}

type ResetUserPasswordRequest struct {
	Token       string `json:"token" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=8,max=100"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type RefreshTokenResponse struct {
	AccessToken string `json:"access_token"`
}
