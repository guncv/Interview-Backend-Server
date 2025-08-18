package repositories

import (
	"mime/multipart"
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

type ResetUserPasswordTxModel struct {
	UserID       string
	PasswordHash string
	ResetToken   string
	UpdatedAt    time.Time
}

type GetResumeJsonWithSummaryDataReq struct {
	SessionID       uuid.UUID
	Position        string
	Company         string
	WorkType        string
	JobRequirements string
	InterviewType   string
	Language        string
	ResumeFile      *multipart.FileHeader
}

type GetResumeJsonWithSummaryDataResponse struct {
	ParsedJson  string `mapstructure:"parsed_json" json:"parsed_json"`
	RawText     string `mapstructure:"raw_text" json:"raw_text"`
	SummaryText string `mapstructure:"summary_text" json:"summary_text"`
}
