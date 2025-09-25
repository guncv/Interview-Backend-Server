package repositories

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	db "gitlab.com/interview-simulation/interview-backend-server/internal/db/sqlc"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
)

type InterviewStateRepository interface {
	CreateInterviewStateWithUpdateFlagSessionTx(ctx context.Context, req *CreateInterviewStateWithUpdateFlagSessionTxReq) error
	EndOldInterviewStateAndCreateNewInterviewStateWithUpdateFlagSessionTx(ctx context.Context, req *EndOldInterviewStateAndCreateNewInterviewStateWithUpdateFlagSessionTxReq) error
}

type interviewStateRepository struct {
	log *log.Logger
	db  db.Store
}

func NewInterviewStateRepository(l *log.Logger, db db.Store) InterviewStateRepository {
	return &interviewStateRepository{
		log: l,
		db:  db,
	}
}

func (r *interviewStateRepository) CreateInterviewStateWithUpdateFlagSessionTx(ctx context.Context, req *CreateInterviewStateWithUpdateFlagSessionTxReq) error {
	r.log.InfoWithID(ctx, "[Repository: CreateInterviewStateWithUpdateFlagSession] Called")

	err := r.db.ExecTx(ctx, func(q *db.Queries) error {

		createInterviewStateReq := db.CreateInterviewStateParams{
			ID:         req.ID,
			SessionID:  req.SessionID,
			PhraseType: req.PhraseType,
			StartedAt:  req.StartedAt,
		}

		if err := q.CreateInterviewState(ctx, createInterviewStateReq); err != nil {
			r.log.ErrorWithID(ctx, "[Repository: CreateInterviewStateWithUpdateFlagSession] Error creating interview state", err)
			return app_error.HandleDatabaseError(err)
		}

		updateInterviewStateReq := db.UpdateCurrentStateAndIDInterviewSessionByIDParams{
			ID:             req.SessionID,
			CurrentState:   sql.NullString{String: req.PhraseType, Valid: true},
			CurrentStateID: uuid.NullUUID{UUID: req.ID, Valid: true},
		}

		rowsAffected, err := q.UpdateCurrentStateAndIDInterviewSessionByID(ctx, updateInterviewStateReq)
		if err != nil {
			r.log.ErrorWithID(ctx, "[Repository: CreateInterviewStateWithUpdateFlagSession] Error creating interview state", err)
			return app_error.HandleDatabaseError(err)
		}

		if rowsAffected == 0 {
			r.log.ErrorWithID(ctx, "[Repository: CreateInterviewStateWithUpdateFlagSession] Error updating interview state", errors.New("rows affected is 0"))
			return app_error.New(errors.New("rows affected is 0"), app_error.ErrCodeSessionNotFound)
		}

		return nil
	})

	if err != nil {
		r.log.ErrorWithID(ctx, "[Repository: CreateInterviewStateWithUpdateFlagSession] Error creating interview state", err)
		return err
	}

	return nil
}

func (r *interviewStateRepository) EndOldInterviewStateAndCreateNewInterviewStateWithUpdateFlagSessionTx(ctx context.Context, req *EndOldInterviewStateAndCreateNewInterviewStateWithUpdateFlagSessionTxReq) error {
	r.log.InfoWithID(ctx, "[Repository: EndOldInterviewStateAndCreateNewInterviewStateWithUpdateFlagSession] Called")

	err := r.db.ExecTx(ctx, func(q *db.Queries) error {

		updateInterviewStateReq := db.UpdateEndedAtInterviewStateByIDParams{
			ID:      req.ID,
			EndedAt: sql.NullTime{Time: req.EndedAt, Valid: true},
		}

		rowsAffected, err := q.UpdateEndedAtInterviewStateByID(ctx, updateInterviewStateReq)
		if err != nil {
			r.log.ErrorWithID(ctx, "[Repository: EndOldInterviewStateAndCreateNewInterviewStateWithUpdateFlagSession] Error updating interview state", err)
			return app_error.HandleDatabaseError(err)
		}

		if rowsAffected == 0 {
			r.log.ErrorWithID(ctx, "[Repository: EndOldInterviewStateAndCreateNewInterviewStateWithUpdateFlagSession] Error updating interview state", errors.New("rows affected is 0"))
			return app_error.New(errors.New("rows affected is 0"), app_error.ErrCodeSessionNotFound)
		}

		createInterviewStateReq := db.CreateInterviewStateParams{
			ID:         req.NewID,
			SessionID:  req.SessionID,
			PhraseType: req.PhraseType,
			StartedAt:  req.StartedAt,
		}

		if err := q.CreateInterviewState(ctx, createInterviewStateReq); err != nil {
			r.log.ErrorWithID(ctx, "[Repository: EndOldInterviewStateAndCreateNewInterviewStateWithUpdateFlagSession] Error creating interview state", err)
			return app_error.HandleDatabaseError(err)
		}

		updateCurrentStateSessionReq := db.UpdateCurrentStateAndIDInterviewSessionByIDParams{
			ID:             req.SessionID,
			CurrentState:   sql.NullString{String: req.PhraseType, Valid: true},
			CurrentStateID: uuid.NullUUID{UUID: req.NewID, Valid: true},
		}

		rowsAffected, err = q.UpdateCurrentStateAndIDInterviewSessionByID(ctx, updateCurrentStateSessionReq)
		if err != nil {
			r.log.ErrorWithID(ctx, "[Repository: EndOldInterviewStateAndCreateNewInterviewStateWithUpdateFlagSession] Error updating current state and id interview session", err)
			return app_error.HandleDatabaseError(err)
		}

		if rowsAffected == 0 {
			r.log.ErrorWithID(ctx, "[Repository: EndOldInterviewStateAndCreateNewInterviewStateWithUpdateFlagSession] Error updating current state and id interview session", errors.New("rows affected is 0"))
			return app_error.New(errors.New("rows affected is 0"), app_error.ErrCodeSessionNotFound)
		}

		return nil
	})

	if err != nil {
		r.log.ErrorWithID(ctx, "[Repository: EndOldInterviewStateAndCreateNewInterviewStateWithUpdateFlagSession] Error updating interview state", err)
		return err
	}
	return nil
}
