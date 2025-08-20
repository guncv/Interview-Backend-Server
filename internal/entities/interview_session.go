package entities

import (
	"mime/multipart"
	"time"

	"github.com/google/uuid"
)

type CreateInterviewSessionWithNewResumeRequest struct {
	File            *multipart.FileHeader `form:"file" binding:"required"`
	Position        string                `form:"position" binding:"required"`
	Company         string                `form:"company" binding:"required"`
	WorkType        string                `form:"work_type" binding:"required"`
	JobRequirements string                `form:"job_requirements" binding:"required"`
	InterviewType   string                `form:"interview_type" binding:"required"`
	Language        string                `form:"language" binding:"required"`
	ConsentAt       time.Time             `form:"consent_at" binding:"required"`
}

type CreateInterviewSessionWithNewResumeResponse struct {
	SessionToken string `json:"session_token"`
}

type CreateInterviewSessionWithExistingResumeReq struct {
	ResumeID        string    `json:"resume_id" binding:"required"`
	Position        string    `json:"position" binding:"required"`
	Company         string    `json:"company" binding:"required"`
	WorkType        string    `json:"work_type" binding:"required"`
	JobRequirements string    `json:"job_requirements" binding:"required"`
	InterviewType   string    `json:"interview_type" binding:"required"`
	Language        string    `json:"language" binding:"required"`
	ConsentAt       time.Time `json:"consent_at" binding:"required"`
}

type CreateInterviewSessionWithExistingResumeResp struct {
	SessionToken string `json:"session_token"`
}

type CreateInterviewSessionTokenReq struct {
	SessionID uuid.UUID     `json:"session_id"`
	UserID    string        `json:"user_id"`
	Duration  time.Duration `json:"duration"`
}

type CreateInterviewSessionTokenResp struct {
	Token string `json:"token"`
}

type DeleteJobRequirementPayload struct {
	JobRequirementID uuid.UUID `json:"job_requirement_id"`
}
