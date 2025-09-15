package repositories

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"gitlab.com/interview-simulation/interview-backend-server/internal/entities"
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
	UserID       uuid.UUID
	PasswordHash string
	ResetToken   string
	UpdatedAt    time.Time
}

type ExtractResumeJsonForRAGReq struct {
	SessionID  string
	UserID     string
	ResumeID   string
	ResumeFile *aws.CustomFileHeader
}

type Experience struct {
	Company     string `mapstructure:"company" json:"company"`
	Position    string `mapstructure:"position" json:"position"`
	JobType     string `mapstructure:"job_type" json:"job_type"`
	StartDate   string `mapstructure:"start_date" json:"start_date"`
	EndDate     string `mapstructure:"end_date" json:"end_date"`
	Description string `mapstructure:"description" json:"description"`
}

type Education struct {
	School       string `mapstructure:"school" json:"school"`
	Degree       string `mapstructure:"degree" json:"degree"`
	FieldOfStudy string `mapstructure:"field_of_study" json:"field_of_study"`
	StartDate    string `mapstructure:"start_date" json:"start_date"`
	EndDate      string `mapstructure:"end_date" json:"end_date"`
	Description  string `mapstructure:"description" json:"description"`
}
type PromptInfo struct {
	FirstName      string       `mapstructure:"first_name" json:"first_name"`
	LastName       string       `mapstructure:"last_name" json:"last_name"`
	Email          string       `mapstructure:"email" json:"email"`
	Phone          string       `mapstructure:"phone" json:"phone"`
	Location       string       `mapstructure:"location" json:"location"`
	Experience     []Experience `mapstructure:"experience" json:"experience"`
	Education      []Education  `mapstructure:"education" json:"education"`
	Skills         []string     `mapstructure:"skills" json:"skills"`
	Certifications []string     `mapstructure:"certifications" json:"certifications"`
	Language       string       `mapstructure:"language" json:"language"`
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

type CreateInterviewSessionTxReq struct {
	ResumeID   uuid.UUID
	UserID     uuid.UUID
	FileName   string
	StorageKey string
	MimeType   string
	ByteSize   int32
	IsDefault  bool

	SessionID uuid.UUID
	Position  string
	Status    string
	Modality  string
	IsConsent bool
}

type InterviewFeedbackAndScoreReq struct {
	UserMessage         string                 `json:"user_message"`
	InterviewerMessage  string                 `json:"interviewer_message"`
	RubricName          string                 `json:"rubric_name"`
	RubricDescriptionMd string                 `json:"rubric_description_md"`
	Criteria            []entities.CritetiaRow `json:"criteria"`
}

type InterviewFeedbackAndScoreResp struct {
	OverallScore    float64         `json:"overall_score"`
	OverallFeedback string          `json:"overall_feedback"`
	CriteriaScores  []CriteriaScore `json:"criteria_scores"`
	ImproveSentence string          `json:"improvement_sentence"`
	LLmModel        string          `json:"llm_model"`
}

type CriteriaScore struct {
	CriterionID       string `json:"criterion_id"`
	CriterionCode     string `json:"criterion_code"`
	CriterionName     string `json:"criterion_name"`
	CriterionScore    int    `json:"criterion_score"`
	CriterionFeedback string `json:"criterion_feedback"`
}

type CreateEvaluationAndScoreTxReq struct {
	EvaluationID uuid.UUID
	SessionID    uuid.UUID
	TurnID       uuid.UUID
	RubricID     uuid.UUID
	UserID       uuid.UUID
	CurrentState string
	OverallScore string
	SummaryMd    string
	CreatedAt    time.Time
	UpdatedAt    time.Time

	ImproveSentenceID uuid.UUID
	ImproveSentence   string
	LLmModel          string

	Criteria []CreateScoreTxReq
}

type CreateScoreTxReq struct {
	ID          uuid.UUID
	CriterionID uuid.UUID
	Score       int
	CommentMd   string
}
