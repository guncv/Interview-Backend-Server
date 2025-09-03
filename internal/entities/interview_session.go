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
	IsConsent       bool                  `form:"is_consent" binding:"required"`
}

type CreateInterviewSessionWithNewResumeResponse struct {
	SessionToken string `json:"session_token"`
}

type CreateInterviewSessionWithExistingResumeReq struct {
	ResumeID        string `json:"resume_id" binding:"required"`
	Position        string `json:"position" binding:"required"`
	Company         string `json:"company" binding:"required"`
	WorkType        string `json:"work_type" binding:"required"`
	JobRequirements string `json:"job_requirements" binding:"required"`
	InterviewType   string `json:"interview_type" binding:"required"`
	Language        string `json:"language" binding:"required"`
	IsConsent       bool   `json:"is_consent" binding:"required"`
}

type CreateInterviewSessionWithExistingResumeResp struct {
	SessionToken string `json:"session_token"`
}

type CreateInterviewSessionTokenResp struct {
	Token string `json:"token"`
}

type DeleteJobRequirementPayload struct {
	JobRequirementID uuid.UUID `json:"job_requirement_id"`
}

type CreateSessionTurnBySessionIDReq struct {
	SessionID  string `json:"session_id" binding:"required"`
	Actor      string `json:"actor" binding:"required"`
	Transcript string `json:"transcript" binding:"required"`
}

type UpdateInterviewSessionStatusReq struct {
	SessionID string `json:"session_id" binding:"required"`
	Status    string `json:"status" binding:"required"`
}

type SetSessionStartTimeReq struct {
	SessionID string  `json:"session_id" binding:"required"`
	StartedAt float64 `json:"started_at" binding:"required"`
}

type SetSessionEndTimeReq struct {
	SessionID string  `json:"session_id" binding:"required"`
	EndedAt   float64 `json:"ended_at" binding:"required"`
}

type IsSessionValidReq struct {
	SessionToken string `json:"session_token" binding:"required"`
	UserID       string `json:"user_id" binding:"required"`
}

type IsSessionValidResp struct {
	UserID    string `json:"user_id"`
	SessionID string `json:"session_id"`
	Language  string `json:"language"`
}

type WebSocketSessionReq struct {
	UserID    string        `json:"user_id"`
	SessionID string        `json:"session_id"`
	Duration  time.Duration `json:"duration"`
}
