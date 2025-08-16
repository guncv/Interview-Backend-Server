package repositories

import (
	"time"

	"github.com/google/uuid"
)

type SignInUserByEmailAndPasswordTxModel struct {
	Email            string
	LastLoginAt      time.Time
	SessionID        uuid.UUID
	UpdatedAt        time.Time
	UserID           uuid.UUID
	UserAgent        string
	IpAddress        string
	RefreshTokenHash string
	LastActive       time.Time
	ExpiresAt        time.Time
}
