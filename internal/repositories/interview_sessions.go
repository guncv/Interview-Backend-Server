package repositories

import (
	"context"
	"errors"

	db "gitlab.com/interview-simulation/interview-backend-server/internal/db/sqlc"
	app_error "gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	log "gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
)

type InterviewSessionRepository interface {
	StartInterviewSession(ctx context.Context, req *db.StartInterviewSessionParams) error
	CreateInterviewSessionWithNewResumeTx(ctx context.Context, req *CreateInterviewSessionTxReq) error
	CreateInterviewSessionWithExistingResumeTx(ctx context.Context, req *CreateInterviewSessionWithExistingResumeTxReq) error
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

func (r *interviewSessionRepository) StartInterviewSession(ctx context.Context, req *db.StartInterviewSessionParams) error {
	r.log.InfoWithID(ctx, "[Repository: StartInterviewSession] Called")

	rowAffected, err := r.db.StartInterviewSession(ctx, *req)
	if err != nil {
		r.log.ErrorWithID(ctx, "[Repository: StartInterviewSession] Error starting interview session", err)
		return app_error.HandleDatabaseError(err)
	}

	if rowAffected == 0 {
		err := errors.New("interview session not found")
		r.log.ErrorWithID(ctx, "[Repository: StartInterviewSession] Interview session not found", err)
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

		if err := q.CreateJobRequirement(ctx, db.CreateJobRequirementParams{
			ID:              req.JobRequirementID,
			UserID:          req.UserID,
			Position:        req.Position,
			CompanyName:     req.CompanyName,
			WorkType:        req.WorkType,
			JobRequirements: req.JobRequirements,
			InterviewType:   req.InterviewType,
			Language:        req.Language,
		}); err != nil {
			r.log.ErrorWithID(ctx, "[Repository: CreateInterviewSessionWithNewResume] Error creating interview session with new resume", err)
			return app_error.HandleDatabaseError(err)
		}

		if err := q.CreateInterviewSession(ctx, db.CreateInterviewSessionParams{
			ID:            req.SessionID,
			UserID:        req.UserID,
			ResumeID:      req.ResumeID,
			RequirementID: req.JobRequirementID,
			PromptJson:    req.PromptJson,
			Status:        req.Status,
			Modality:      req.Modality,
			IsConsent:     req.IsConsent,
		}); err != nil {
			r.log.ErrorWithID(ctx, "[Repository: CreateInterviewSessionWithNewResume] Error creating interview session with new resume", err)
			return app_error.HandleDatabaseError(err)
		}

		return nil
	})

	if err != nil {
		r.log.ErrorWithID(ctx, "[Repository: CreateInterviewSessionWithNewResume] Error creating interview session with new resume", err)
		return app_error.HandleDatabaseError(err)
	}

	return nil
}

func (r *interviewSessionRepository) CreateInterviewSessionWithExistingResumeTx(ctx context.Context, req *CreateInterviewSessionWithExistingResumeTxReq) error {
	r.log.InfoWithID(ctx, "[Repository: CreateInterviewSessionWithExistingResume] Called")

	err := r.db.ExecTx(ctx, func(q *db.Queries) error {
		if err := q.CreateJobRequirement(ctx, db.CreateJobRequirementParams{
			ID:              req.JobRequirementID,
			UserID:          req.UserID,
			Position:        req.Position,
			CompanyName:     req.CompanyName,
			WorkType:        req.WorkType,
			JobRequirements: req.JobRequirements,
			InterviewType:   req.InterviewType,
			Language:        req.Language,
		}); err != nil {
			r.log.ErrorWithID(ctx, "[Repository: CreateInterviewSessionWithExistingResume] Error creating interview session with existing resume", err)
			return app_error.HandleDatabaseError(err)
		}

		if err := q.CreateInterviewSession(ctx, db.CreateInterviewSessionParams{
			ID:            req.SessionID,
			UserID:        req.UserID,
			ResumeID:      req.ResumeID,
			RequirementID: req.JobRequirementID,
			PromptJson:    req.PromptJson,
			Status:        req.Status,
			Modality:      req.Modality,
			IsConsent:     req.IsConsent,
		}); err != nil {
			r.log.ErrorWithID(ctx, "[Repository: CreateInterviewSessionWithExistingResume] Error creating interview session with existing resume", err)
			return app_error.HandleDatabaseError(err)
		}

		return nil
	})

	if err != nil {
		r.log.ErrorWithID(ctx, "[Repository: CreateInterviewSessionWithExistingResume] Error creating interview session with existing resume", err)
		return app_error.HandleDatabaseError(err)
	}

	return nil
}
