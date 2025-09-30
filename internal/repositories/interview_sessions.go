package repositories

import (
	"context"
	"database/sql"
	"errors"

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
	UpdateStartedAtInterviewSession(ctx context.Context, req *db.UpdateStartedAtInterviewSessionParams) error
	GetSessionState(ctx context.Context, sessionID uuid.UUID) (*db.GetSessionStateRow, error)
	UpdateIsStartedConversationSession(ctx context.Context, req *db.UpdateIsStartedConversationSessionParams) error
	ListInterviewSessionsByUserIDFirstPage(ctx context.Context, req *db.ListInterviewSessionsByUserIDFirstPageParams) ([]db.ListInterviewSessionsByUserIDFirstPageRow, error)
	ListInterviewSessionsByUserIDWithCursor(ctx context.Context, req *db.ListInterviewSessionsByUserIDWithCursorParams) ([]db.ListInterviewSessionsByUserIDWithCursorRow, error)
	ListInterviewSessionsByUserIDWithJumpPagination(ctx context.Context, req *db.ListInterviewSessionsByUserIDWithJumpPaginationParams) ([]db.ListInterviewSessionsByUserIDWithJumpPaginationRow, error)
	CountInterviewSessionsByUserID(ctx context.Context, req *db.CountInterviewSessionsByUserIDParams) (int64, error)
	DeleteUserInterviewSessionByID(ctx context.Context, req *db.DeleteUserInterviewSessionByIDParams) error
	GetInterviewSessionInformationByID(ctx context.Context, sessionID uuid.UUID) (*db.GetInterviewSessionInformationByIDRow, error)
	UpdateFinalizeStatusInterviewSessionByID(ctx context.Context, req *db.UpdateFinalizeStatusInterviewSessionByIDParams) error
	GetInterviewSessionStatusByID(ctx context.Context, sessionID uuid.UUID) (string, error)
	UpdateIsTimedOutSession(ctx context.Context, req *db.UpdateIsTimedOutSessionParams) error
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

func (r *interviewSessionRepository) GetSessionState(ctx context.Context, sessionID uuid.UUID) (*db.GetSessionStateRow, error) {
	r.log.InfoWithID(ctx, "[Repository: GetStartedAndIsStartedConversationSession] Called")

	resp, err := r.db.GetSessionState(ctx, sessionID)
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

func (r *interviewSessionRepository) ListInterviewSessionsByUserIDFirstPage(ctx context.Context, req *db.ListInterviewSessionsByUserIDFirstPageParams) ([]db.ListInterviewSessionsByUserIDFirstPageRow, error) {
	r.log.InfoWithID(ctx, "[Repository: ListInterviewSessionsByUserIDFirstPage] Called: ", req)

	resp, err := r.db.ListInterviewSessionsByUserIDFirstPage(ctx, *req)
	if err != nil {
		r.log.ErrorWithID(ctx, "[Repository: ListInterviewSessionsByUserIDFirstPage] Error listing interview sessions by user ID first page", err)
		return nil, app_error.HandleDatabaseError(err)
	}

	return resp, nil
}

func (r *interviewSessionRepository) ListInterviewSessionsByUserIDWithCursor(ctx context.Context, req *db.ListInterviewSessionsByUserIDWithCursorParams) ([]db.ListInterviewSessionsByUserIDWithCursorRow, error) {
	r.log.InfoWithID(ctx, "[Repository: ListInterviewSessionsByUserIDWithCursor] Called: ", req)

	resp, err := r.db.ListInterviewSessionsByUserIDWithCursor(ctx, *req)
	if err != nil {
		r.log.ErrorWithID(ctx, "[Repository: ListInterviewSessionsByUserIDWithCursor] Error listing interview sessions by user ID with cursor", err)
		return nil, app_error.HandleDatabaseError(err)
	}

	return resp, nil
}

func (r *interviewSessionRepository) ListInterviewSessionsByUserIDWithJumpPagination(ctx context.Context, req *db.ListInterviewSessionsByUserIDWithJumpPaginationParams) ([]db.ListInterviewSessionsByUserIDWithJumpPaginationRow, error) {
	r.log.InfoWithID(ctx, "[Repository: ListInterviewSessionsByUserIDWithJumpPagination] Called: ", req)

	resp, err := r.db.ListInterviewSessionsByUserIDWithJumpPagination(ctx, *req)
	if err != nil {
		r.log.ErrorWithID(ctx, "[Repository: ListInterviewSessionsByUserIDWithJumpPagination] Error listing interview sessions by user ID with jump pagination", err)
		return nil, app_error.HandleDatabaseError(err)
	}

	return resp, nil
}

func (r *interviewSessionRepository) CountInterviewSessionsByUserID(ctx context.Context, req *db.CountInterviewSessionsByUserIDParams) (int64, error) {
	r.log.InfoWithID(ctx, "[Repository: CountInterviewSessionsByUserID] Called")

	count, err := r.db.CountInterviewSessionsByUserID(ctx, *req)
	if err != nil {
		r.log.ErrorWithID(ctx, "[Repository: CountInterviewSessionsByUserID] Error counting interview sessions by user ID", err)
		return 0, app_error.HandleDatabaseError(err)
	}

	return count, nil
}

func (r *interviewSessionRepository) DeleteUserInterviewSessionByID(ctx context.Context, req *db.DeleteUserInterviewSessionByIDParams) error {
	r.log.InfoWithID(ctx, "[Repository: DeleteUserInterviewSessionByID] Called")

	rowAffected, err := r.db.DeleteUserInterviewSessionByID(ctx, *req)
	if err != nil {
		r.log.ErrorWithID(ctx, "[Repository: DeleteUserInterviewSessionByID] Error deleting interview session", err)
		return app_error.HandleDatabaseError(err)
	}

	if rowAffected == 0 {
		err := errors.New("interview session not found")
		r.log.ErrorWithID(ctx, "[Repository: DeleteUserInterviewSessionByID] Interview session not found", err)
		return app_error.New(err, app_error.ErrCodeSessionNotFound)
	}

	return nil
}

func (r *interviewSessionRepository) GetInterviewSessionInformationByID(ctx context.Context, sessionID uuid.UUID) (*db.GetInterviewSessionInformationByIDRow, error) {
	r.log.InfoWithID(ctx, "[Repository: GetInterviewSessionInformationByID] Called")

	resp, err := r.db.GetInterviewSessionInformationByID(ctx, sessionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.log.ErrorWithID(ctx, "[Repository: GetInterviewSessionInformationByID] Interview session not found", err)
			return nil, app_error.New(err, app_error.ErrCodeSessionNotFound)
		}
		r.log.ErrorWithID(ctx, "[Repository: GetInterviewSessionInformationByID] Error getting interview session information", err)
		return nil, app_error.HandleDatabaseError(err)
	}
	return &resp, nil
}

func (r *interviewSessionRepository) UpdateFinalizeStatusInterviewSessionByID(ctx context.Context, req *db.UpdateFinalizeStatusInterviewSessionByIDParams) error {
	r.log.InfoWithID(ctx, "[Repository: UpdateFinalizeStatusInterviewSessionByID] Called")

	rowAffected, err := r.db.UpdateFinalizeStatusInterviewSessionByID(ctx, *req)
	if err != nil {
		r.log.ErrorWithID(ctx, "[Repository: UpdateFinalizeStatusInterviewSessionByID] Error updating interview session finalize status", err)
		return app_error.HandleDatabaseError(err)
	}

	if rowAffected == 0 {
		r.log.ErrorWithID(ctx, "[Repository: UpdateFinalizeStatusInterviewSessionByID] Interview session not found")
		return app_error.New(constants.ErrInterviewSessionNotFound, app_error.ErrCodeSessionNotFound)
	}

	return nil
}

func (r *interviewSessionRepository) GetInterviewSessionStatusByID(ctx context.Context, sessionID uuid.UUID) (string, error) {
	r.log.InfoWithID(ctx, "[Repository: GetInterviewSessionStatusByID] Called")

	resp, err := r.db.GetInterviewSessionStatusByID(ctx, sessionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.log.ErrorWithID(ctx, "[Repository: GetInterviewSessionStatusByID] Interview session not found", err)
			return "", app_error.New(constants.ErrInterviewSessionNotFound, app_error.ErrCodeSessionNotFound)
		}
		r.log.ErrorWithID(ctx, "[Repository: GetInterviewSessionStatusByID] Error getting interview session status", err)
		return "", app_error.HandleDatabaseError(err)
	}
	return resp, nil
}

func (r *interviewSessionRepository) UpdateIsTimedOutSession(ctx context.Context, req *db.UpdateIsTimedOutSessionParams) error {
	r.log.InfoWithID(ctx, "[Repository: UpdateIsTimedOutSession] Called")

	rowAffected, err := r.db.UpdateIsTimedOutSession(ctx, *req)
	if err != nil {
		r.log.ErrorWithID(ctx, "[Repository: UpdateIsTimedOutSession] Error updating interview session is timed out", err)
		return app_error.HandleDatabaseError(err)
	}

	if rowAffected == 0 {
		err := errors.New("interview session not found")
		r.log.ErrorWithID(ctx, "[Repository: UpdateIsTimedOutSession] Interview session not found", err)
		return app_error.New(err, app_error.ErrCodeSessionNotFound)
	}

	return nil
}
