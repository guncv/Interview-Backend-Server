package entities

import (
	"mime/multipart"
	"time"

	"github.com/google/uuid"
)

type CreateInterviewSessionWithNewResumeRequest struct {
	File      *multipart.FileHeader `form:"file" binding:"required"`
	Position  string                `form:"position" binding:"required"`
	IsConsent bool                  `form:"is_consent" binding:"required"`
}

type CreateInterviewSessionWithNewResumeResponse struct {
	SessionToken string `json:"session_token"`
}

type CreateInterviewSessionWithExistingResumeReq struct {
	ResumeID  string `json:"resume_id" binding:"required"`
	Position  string `json:"position" binding:"required"`
	IsConsent bool   `json:"is_consent" binding:"required"`
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

type CreateUserSessionTurnBySessionIDReq struct {
	TurnID       string `json:"turn_id" binding:"required"`
	SessionID    string `json:"session_id" binding:"required"`
	CurrentState string `json:"current_state" binding:"required"`
	Transcript   string `json:"transcript" binding:"required"`
}

type CreateInterviewerSessionTurnBySessionIDReq struct {
	TurnID       string `json:"turn_id" binding:"required"`
	SessionID    string `json:"session_id" binding:"required"`
	Transcript   string `json:"transcript" binding:"required"`
	StartedAt    string `json:"started_at" binding:"required"`
	EndedAt      string `json:"ended_at" binding:"required"`
	CurrentState string `json:"current_state" binding:"required"`
}

type UpdateInterviewSessionStatusReq struct {
	SessionID string `json:"session_id" binding:"required"`
	Status    string `json:"status" binding:"required"`
}

type SetSessionStartTimeReq struct {
	SessionID string `json:"session_id" binding:"required"`
	StartedAt string `json:"started_at" binding:"required"`
}

type SetSessionEndTimeReq struct {
	SessionID string `json:"session_id" binding:"required"`
	EndedAt   string `json:"ended_at" binding:"required"`
}

type IsSessionValidReq struct {
	SessionToken string `json:"session_token" binding:"required"`
	UserID       string `json:"user_id" binding:"required"`
}

type CalculateTurnScoreReq struct {
	SessionID          string `json:"session_id" binding:"required"`
	UserTurnID         string `json:"user_turn_id" binding:"required"`
	UserID             string `json:"user_id" binding:"required"`
	UserMessage        string `json:"user_message" binding:"required"`
	InterviewerMessage string `json:"interviewer_message" binding:"required"`
	CurrentState       string `json:"current_state" binding:"required"`
}

type GetInterviewerLastMessageReq struct {
	SessionID string `json:"session_id" binding:"required"`
}

type GetInterviewerLastMessageResp struct {
	Message      string `json:"message"`
	CurrentState string `json:"current_state"`
}

type CalculateEvaluationInOldStateReq struct {
	SessionID        string `json:"session_id" binding:"required"`
	CurrentState     string `json:"current_state" binding:"required"`
	InterviewStateID string `json:"interview_state_id" binding:"required"`
}

type IsSessionValidResp struct {
	UserID    string `json:"user_id"`
	SessionID string `json:"session_id"`
	ResumeID  string `json:"resume_id"`
}

type WebSocketSessionReq struct {
	UserID    string        `json:"user_id"`
	SessionID string        `json:"session_id"`
	ResumeID  string        `json:"resume_id"`
	Duration  time.Duration `json:"duration"`
}

type RedisLastMessagePayload struct {
	Message      string `json:"message"`
	CurrentState string `json:"current_state"`
}

type GetChatHistoryBySessionTokenReq struct {
	SessionToken string `json:"session_token" binding:"required"`
}

type GetChatHistoryBySessionTokenResp struct {
	ChatHistory []ChatHistory `json:"chat_history"`
}

type ChatHistory struct {
	ID             uuid.UUID `json:"id"`
	TurnNo         int64     `json:"turn_no"`
	Actor          string    `json:"actor"`
	TranscriptText string    `json:"transcript_text"`
	StartAt        string    `json:"start_at"`
	EndAt          string    `json:"end_at"`
	CreatedAt      string    `json:"created_at"`
}

type GetInterviewSessionInformationReq struct {
	SessionToken string `json:"session_token" binding:"required"`
}

type GetInterviewSessionInformationResp struct {
	ResumeID            uuid.UUID `json:"resume_id"`
	ResumeFileName      string    `json:"resume_file_name"`
	Position            string    `json:"position"`
	Status              string    `json:"status"`
	StatusDisplayName   string    `json:"status_display_name"`
	StatusColor         string    `json:"status_color"`
	StartedAt           string    `json:"started_at"`
	EndedAt             string    `json:"ended_at"`
	OverallScore        float64   `json:"overall_score"`
	OverallScorePercent string    `json:"overall_score_percent"`
	OverallScoreColor   string    `json:"overall_score_color"`
	SummaryMd           string    `json:"summary_md"`
	CreatedAt           string    `json:"created_at"`
	CreatedAtFullName   string    `json:"created_at_full_name"`
}

type CheckExistsAndInitStartedAtInterviewSessionResp struct {
	StartedAt             string `json:"started_at"`
	IsStartedConversation bool   `json:"is_started_conversation"`
	CurrentState          string `json:"current_state"`
	CurrentStateID        string `json:"current_state_id"`
}

type EndInterviewSessionReq struct {
	SessionId string `json:"session_id" binding:"required"`
	Status    string `json:"status" binding:"required"`
}

type ListInterviewSessionsByUserIDWithCursorReq struct {
	SearchText *string `form:"search_text"`
	Status     *string `form:"status"`
	Cursor     *Cursor `form:"cursor"`
	Limit      *int    `form:"limit"`
	Type       *string `form:"type"`
}

type ListInterviewSessionsByUserIDWithJumpPaginationReq struct {
	SearchText *string `form:"search_text"`
	Status     *string `form:"status"`
	Offset     *int    `form:"offset"`
	Limit      *int    `form:"limit"`
}

type Cursor struct {
	CreatedAt string `json:"created_at"`
	ID        string `json:"id"`
}

type ListInterviewSessionsByUserIDResp struct {
	Sessions   []InterviewSessionSummary `json:"sessions"`
	PrevCursor *Cursor                   `json:"prev_cursor"`
	NextCursor *Cursor                   `json:"next_cursor"`
	TotalPages int                       `json:"total_pages"`
	PageSize   int                       `json:"page_size"`
}

type InterviewSessionSummary struct {
	ID               string  `json:"id"`
	ResumeID         string  `json:"resume_id"`
	ResumeFileName   string  `json:"resume_file_name"`
	Position         string  `json:"position"`
	Status           string  `json:"status"`
	TotalTime        string  `json:"total_time"`
	OverallScore     float64 `json:"overall_score"`
	CreatedAt        string  `json:"created_at"`
	CreatedAtDisplay string  `json:"created_at_display"`
}

type GetChatHistoryBySessionIDWithEvaluationReq struct {
	SessionID string `json:"session_id" binding:"required"`
	TurnNo    *int32 `json:"turn_no" binding:"required"`
}

type GetChatHistoryBySessionIDWithEvaluationResp struct {
	ChatHistory    []ChatHistoryWithEvaluation `json:"chat_history"`
	CursorTurnNext int32                       `json:"cursor_turn_next"`
}

type ChatHistoryWithEvaluation struct {
	ID                string      `json:"id"`
	TurnNo            int64       `json:"turn_no"`
	Actor             string      `json:"actor"`
	Content           string      `json:"content"`
	StartAt           string      `json:"start_at"`
	EndAt             string      `json:"end_at"`
	Evaluation        *Evaluation `json:"evaluation"`
	CorrectedSentence *string     `json:"corrected_sentence"`
	CurrentState      string      `json:"current_state"`
	CurrentStateColor string      `json:"current_state_color"`
}

type Evaluation struct {
	OverallScore string          `json:"overall_score"`
	OverallColor string          `json:"overall_color"`
	SummaryMd    string          `json:"summary_md"`
	Scores       []CriteriaScore `json:"scores"`
}

type CriteriaScore struct {
	CriterionID   string `json:"criterion_id"`
	CriterionName string `json:"criterion_name"`
	Score         string `json:"score"`
	ScoreColor    string `json:"score_color"`
	CommentMd     string `json:"comment_md"`
}

type InitialFirstCurrentStateSessionReq struct {
	SessionID    string `json:"session_id" binding:"required"`
	CurrentState string `json:"current_state" binding:"required"`
}

type InitialFirstCurrentStateSessionResp struct {
	CurrentState   string `json:"current_state"`
	CurrentStateID string `json:"current_state_id"`
}

type UpdateCurrentStateSessionReq struct {
	OldCurrentStateID string `json:"old_current_state_id" binding:"required"`
	SessionID         string `json:"session_id" binding:"required"`
	CurrentState      string `json:"current_state" binding:"required"`
}

type UpdateCurrentStateSessionResp struct {
	CurrentState   string `json:"current_state"`
	CurrentStateID string `json:"current_state_id"`
}
