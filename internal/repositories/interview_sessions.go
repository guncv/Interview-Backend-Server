package repositories

import (
	"context"
	"errors"

	"github.com/google/uuid"
	db "gitlab.com/interview-simulation/interview-backend-server/internal/db/sqlc"
	app_error "gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	log "gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
)

type InterviewSessionRepository interface {
	CheckInterviewSessionExists(ctx context.Context, sessionID uuid.UUID) (bool, error)
	UpdateInterviewSessionStatus(ctx context.Context, req *db.UpdateInterviewSessionStatusParams) error
	GetMaxTurnNoBySessionID(ctx context.Context, sessionID uuid.UUID) (int64, error)
	CreateSessionTurnBySessionID(ctx context.Context, req *db.CreateInterviewTurnParams) error
	EndInterviewSession(ctx context.Context, req *db.EndInterviewSessionParams) error
	CreateInterviewSessionWithNewResumeTx(ctx context.Context, req *CreateInterviewSessionTxReq) error
	CreateInterviewSession(ctx context.Context, req *db.CreateInterviewSessionParams) error
}

type interviewSessionRepository struct {
	log *log.Logger
	db  db.Store
}

func NewInterviewSessionRepository(
	log *log.Logger,
	db db.Store,
) InterviewSessionRepository {
	return &interviewSessionRepository{
		log: log,
		db:  db,
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

func (r *interviewSessionRepository) GetMaxTurnNoBySessionID(ctx context.Context, sessionID uuid.UUID) (int64, error) {
	r.log.InfoWithID(ctx, "[Repository: GetMaxTurnNoBySessionID] Called")

	turnNo, err := r.db.GetMaxTurnNoBySessionID(ctx, sessionID)
	if err != nil {
		r.log.ErrorWithID(ctx, "[Repository: GetMaxTurnNoBySessionID] Error getting max turn no by session ID", err)
		return 0, app_error.HandleDatabaseError(err)
	}

	return turnNo.(int64), nil
}

func (r *interviewSessionRepository) CreateSessionTurnBySessionID(ctx context.Context, req *db.CreateInterviewTurnParams) error {
	r.log.InfoWithID(ctx, "[Repository: CreateSessionTurnBySessionID] Called")

	if err := r.db.CreateInterviewTurn(ctx, *req); err != nil {
		r.log.ErrorWithID(ctx, "[Repository: CreateSessionTurnBySessionID] Error creating session turn by session ID", err)
		return app_error.HandleDatabaseError(err)
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
			ID:        req.SessionID,
			UserID:    req.UserID,
			ResumeID:  req.ResumeID,
			Position:  req.Position,
			Status:    req.Status,
			Modality:  req.Modality,
			IsConsent: req.IsConsent,
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

	if err := r.db.CreateInterviewSession(ctx, db.CreateInterviewSessionParams{
		ID:        req.ID,
		UserID:    req.UserID,
		ResumeID:  req.ResumeID,
		Position:  req.Position,
		Status:    req.Status,
		Modality:  req.Modality,
		IsConsent: req.IsConsent,
	}); err != nil {
		r.log.ErrorWithID(ctx, "[Repository: CreateInterviewSessionWithExistingResume] Error creating interview session with existing resume", err)
		return app_error.HandleDatabaseError(err)
	}

	return nil
}
