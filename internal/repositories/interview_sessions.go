package repositories

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"
	"gitlab.com/interview-simulation/interview-backend-server/internal/config"
	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
	db "gitlab.com/interview-simulation/interview-backend-server/internal/db/sqlc"
	app_error "gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	log "gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
)

type InterviewSessionRepository interface {
	CheckInterviewSessionExists(ctx context.Context, sessionID uuid.UUID) (bool, error)
	UpdateInterviewSessionStatus(ctx context.Context, req *db.UpdateInterviewSessionStatusParams) error
	EndInterviewSession(ctx context.Context, req *db.EndInterviewSessionParams) error
	CreateInterviewSessionWithNewResumeTx(ctx context.Context, req *CreateInterviewSessionTxReq) error
	CreateInterviewSession(ctx context.Context, req *db.CreateInterviewSessionParams) error
	InterviewFeedbackAndScore(ctx context.Context, req *InterviewFeedbackAndScoreReq) (*InterviewFeedbackAndScoreResp, error)
	GetInterviewSessionInformation(ctx context.Context, sessionID uuid.UUID) (*db.GetInterviewSessionInformationRow, error)
	UpdateStartedAtInterviewSession(ctx context.Context, req *db.UpdateStartedAtInterviewSessionParams) error
	GetStartedAndIsStartedConversationSession(ctx context.Context, sessionID uuid.UUID) (*db.GetStartedAndIsStartedConversationSessionRow, error)
	UpdateIsStartedConversationSession(ctx context.Context, req *db.UpdateIsStartedConversationSessionParams) error
	ListInterviewSessionsByUserID(ctx context.Context, req *db.ListInterviewSessionsByUserIDParams) ([]db.ListInterviewSessionsByUserIDRow, error)
}

type interviewSessionRepository struct {
	log *log.Logger
	db  db.Store
	cfg *config.Config
}

func NewInterviewSessionRepository(
	log *log.Logger,
	db db.Store,
	cfg *config.Config,
) InterviewSessionRepository {
	return &interviewSessionRepository{
		log: log,
		db:  db,
		cfg: cfg,
	}
}

func (r *interviewSessionRepository) CheckInterviewSessionExists(ctx context.Context, sessionID uuid.UUID) (bool, error) {
	r.log.InfoWithID(ctx, "[Repository: CheckInterviewSessionExists] Called")

	exists, err := r.db.CheckInterviewSessionExists(ctx, sessionID)
	if err != nil {
		r.log.ErrorWithID(ctx, "[Repository: CheckInterviewSessionExists] Error checking interview session exists", err)
		return false, app_error.HandleDatabaseError(err)
	}

	return exists, nil
}

func (r *interviewSessionRepository) UpdateInterviewSessionStatus(ctx context.Context, req *db.UpdateInterviewSessionStatusParams) error {
	r.log.InfoWithID(ctx, "[Repository: UpdateInterviewSessionStatus] Called")

	rowAffected, err := r.db.UpdateInterviewSessionStatus(ctx, *req)
	if err != nil {
		r.log.ErrorWithID(ctx, "[Repository: UpdateInterviewSessionStatus] Error updating interview session status", err)
		return app_error.HandleDatabaseError(err)
	}

	if rowAffected == 0 {
		err := errors.New("interview session not found")
		r.log.ErrorWithID(ctx, "[Repository: UpdateInterviewSessionStatus] Interview session not found", err)
		return app_error.New(err, app_error.ErrCodeSessionNotFound)
	}

	return nil
}

func (r *interviewSessionRepository) EndInterviewSession(ctx context.Context, req *db.EndInterviewSessionParams) error {
	r.log.InfoWithID(ctx, "[Repository: EndInterviewSession] Called")

	rowAffected, err := r.db.EndInterviewSession(ctx, *req)
	if err != nil {
		r.log.ErrorWithID(ctx, "[Repository: EndInterviewSession] Error ending interview session", err)
		return app_error.HandleDatabaseError(err)
	}

	if rowAffected == 0 {
		err := errors.New("interview session not found")
		r.log.ErrorWithID(ctx, "[Repository: EndInterviewSession] Interview session not found", err)
		return app_error.New(err, app_error.ErrCodeSessionNotFound)
	}

	return nil
}

func (r *interviewSessionRepository) CreateInterviewSessionWithNewResumeTx(ctx context.Context, req *CreateInterviewSessionTxReq) error {
	r.log.InfoWithID(ctx, "[Repository: CreateInterviewSessionWithNewResume] Called")

	err := r.db.ExecTx(ctx, func(q *db.Queries) error {
		if err := q.CreateResume(ctx, db.CreateResumeParams{
			ID:         req.ResumeID,
			UserID:     req.UserID,
			FileName:   req.FileName,
			StorageKey: req.StorageKey,
			MimeType:   req.MimeType,
			ByteSize:   req.ByteSize,
			IsDefault:  req.IsDefault,
		}); err != nil {
			r.log.ErrorWithID(ctx, "[Repository: CreateInterviewSessionWithNewResume] Error creating interview session with new resume", err)
			return app_error.HandleDatabaseError(err)
		}

		if err := q.CreateInterviewSession(ctx, db.CreateInterviewSessionParams{
			ID:             req.SessionID,
			UserID:         req.UserID,
			ResumeID:       req.ResumeID,
			ResumeFileName: req.FileName,
			Position:       req.Position,
			Status:         req.Status,
			Modality:       req.Modality,
			IsConsent:      req.IsConsent,
		}); err != nil {
			r.log.ErrorWithID(ctx, "[Repository: CreateInterviewSessionWithNewResume] Error creating interview session with new resume", err)
			return app_error.HandleDatabaseError(err)
		}

		return nil
	})

	if err != nil {
		r.log.ErrorWithID(ctx, "[Repository: CreateInterviewSessionWithNewResume] Error creating interview session with new resume", err)
		return err
	}

	return nil
}

func (r *interviewSessionRepository) CreateInterviewSession(ctx context.Context, req *db.CreateInterviewSessionParams) error {
	r.log.InfoWithID(ctx, "[Repository: CreateInterviewSessionWithExistingResume] Called")

	if err := r.db.CreateInterviewSession(ctx, *req); err != nil {
		r.log.ErrorWithID(ctx, "[Repository: CreateInterviewSessionWithExistingResume] Error creating interview session with existing resume", err)
		return app_error.HandleDatabaseError(err)
	}

	return nil
}

func (r *interviewSessionRepository) InterviewFeedbackAndScore(ctx context.Context, req *InterviewFeedbackAndScoreReq) (*InterviewFeedbackAndScoreResp, error) {
	r.log.InfoWithID(ctx, "[Repository: InterviewFeedbackAndScore] Called")

	endpoint := r.cfg.InterviewSessionConfig.InterviewAgentURL + constants.PathFeedbackAndScoreAgent

	jsonBody, err := json.Marshal(req)
	if err != nil {
		r.log.ErrorWithID(ctx, "[Repository: InterviewFeedbackAndScore] Failed to marshal request body", err)
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", endpoint, bytes.NewReader(jsonBody))
	if err != nil {
		r.log.ErrorWithID(ctx, "[Repository: InterviewFeedbackAndScore] Failed to create HTTP request", err)
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: constants.TimeoutHTTP}
	resp, err := client.Do(httpReq)
	if err != nil {
		r.log.ErrorWithID(ctx, "[Repository: InterviewFeedbackAndScore] HTTP request failed", err)
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		r.log.ErrorWithID(ctx, "[Repository: InterviewFeedbackAndScore] Failed to read response body", err)
		return nil, err
	}

	r.log.InfoWithID(ctx, fmt.Sprintf("[Repository: InterviewFeedbackAndScore] Response: %s | Body: %s", resp.Status, string(bodyBytes)))

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("interview agent returned status: %s | body: %s", resp.Status, string(bodyBytes))
		r.log.ErrorWithID(ctx, "[Repository: InterviewFeedbackAndScore] HTTP error response", err)
		return nil, err
	}

	var result InterviewFeedbackAndScoreResp
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		r.log.ErrorWithID(ctx, "[Repository: InterviewFeedbackAndScore] Failed to unmarshal response body", err)
		return nil, err
	}

	return &result, nil
}

func (r *interviewSessionRepository) GetInterviewSessionInformation(ctx context.Context, sessionID uuid.UUID) (*db.GetInterviewSessionInformationRow, error) {
	r.log.InfoWithID(ctx, "[Repository: GetInterviewSessionInformation] Called")

	session, err := r.db.GetInterviewSessionInformation(ctx, sessionID)
	if err != nil {
		r.log.ErrorWithID(ctx, "[Repository: GetInterviewSessionInformation] Error getting interview session information", err)
		return nil, app_error.HandleDatabaseError(err)
	}
	return &session, nil
}

func (r *interviewSessionRepository) UpdateStartedAtInterviewSession(ctx context.Context, req *db.UpdateStartedAtInterviewSessionParams) error {
	r.log.InfoWithID(ctx, "[Repository: UpdateStartedAtInterviewSession] Called")

	rowAffected, err := r.db.UpdateStartedAtInterviewSession(ctx, db.UpdateStartedAtInterviewSessionParams{
		ID:        req.ID,
		StartedAt: req.StartedAt,
	})
	if err != nil {
		r.log.ErrorWithID(ctx, "[Repository: UpdateStartedAtInterviewSession] Error updating interview session started at", err)
		return app_error.HandleDatabaseError(err)
	}

	if rowAffected == 0 {
		err := errors.New("interview session not found")
		r.log.ErrorWithID(ctx, "[Repository: UpdateStartedAtInterviewSession] Interview session not found", err)
		return app_error.New(err, app_error.ErrCodeSessionNotFound)
	}

	return nil
}

func (r *interviewSessionRepository) GetStartedAndIsStartedConversationSession(ctx context.Context, sessionID uuid.UUID) (*db.GetStartedAndIsStartedConversationSessionRow, error) {
	r.log.InfoWithID(ctx, "[Repository: GetStartedAndIsStartedConversationSession] Called")

	resp, err := r.db.GetStartedAndIsStartedConversationSession(ctx, sessionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.log.ErrorWithID(ctx, "[Repository: GetStartedAndIsStartedConversationSession] Started at interview session not found", err)
			return nil, app_error.New(err, app_error.ErrCodeSessionNotFound)
		}
		r.log.ErrorWithID(ctx, "[Repository: GetStartedAndIsStartedConversationSession] Error getting started at interview session", err)
		return nil, app_error.HandleDatabaseError(err)
	}

	return &resp, nil
}

func (r *interviewSessionRepository) UpdateIsStartedConversationSession(ctx context.Context, req *db.UpdateIsStartedConversationSessionParams) error {
	r.log.InfoWithID(ctx, "[Repository: UpdateIsStartedConversationSession] Called")

	rowAffected, err := r.db.UpdateIsStartedConversationSession(ctx, *req)
	if err != nil {
		r.log.ErrorWithID(ctx, "[Repository: UpdateIsStartedConversationSession] Error updating interview session is started conversation", err)
		return app_error.HandleDatabaseError(err)
	}

	if rowAffected == 0 {
		err := errors.New("interview session not found")
		r.log.ErrorWithID(ctx, "[Repository: UpdateIsStartedConversationSession] Interview session not found", err)
		return app_error.New(err, app_error.ErrCodeSessionNotFound)
	}

	return nil
}

func (r *interviewSessionRepository) ListInterviewSessionsByUserID(ctx context.Context, req *db.ListInterviewSessionsByUserIDParams) ([]db.ListInterviewSessionsByUserIDRow, error) {
	r.log.InfoWithID(ctx, "[Repository: ListInterviewSessionsByUserID] Called")

	resp, err := r.db.ListInterviewSessionsByUserID(ctx, *req)
	if err != nil {
		r.log.ErrorWithID(ctx, "[Repository: ListInterviewSessionsByUserID] Error listing interview sessions by user ID", err)
		return nil, app_error.HandleDatabaseError(err)
	}

	return resp, nil
}
