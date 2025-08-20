package repositories

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/aws"
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
	ResumeFile      *aws.CustomFileHeader
}

type GetResumeJsonWithSummaryDataResponse struct {
	ParsedJson PromptInfo `mapstructure:"parsed_json" json:"parsed_json"`
}

type PromptInfo struct {
	FullName       string   `mapstructure:"full_name" json:"full_name"`
	Email          string   `mapstructure:"email" json:"email"`
	Phone          string   `mapstructure:"phone" json:"phone"`
	Location       string   `mapstructure:"location" json:"location"`
	Experience     []string `mapstructure:"experience" json:"experience"`
	Education      []string `mapstructure:"education" json:"education"`
	Skills         []string `mapstructure:"skills" json:"skills"`
	Certifications []string `mapstructure:"certifications" json:"certifications"`
	Language       string   `mapstructure:"language" json:"language"`
}

type CreateResumeAndJobRequirementReq struct {
	ResumeID   uuid.UUID
	UserID     uuid.UUID
	FileName   string
	StorageKey string
	MimeType   string
	ByteSize   int32
	IsDefault  bool

	JobRequirementID uuid.UUID
	Position         string
	CompanyName      string
	WorkType         string
	JobRequirements  string
	InterviewType    string
	Language         string
	CreatedAt        sql.NullTime
	UpdatedAt        sql.NullTime
}
