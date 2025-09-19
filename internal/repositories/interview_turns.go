package repositories

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	db "gitlab.com/interview-simulation/interview-backend-server/internal/db/sqlc"
	app_error "gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	log "gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
)

type InterviewTurnsRepository interface {
	GetInterviewerLastMessage(ctx context.Context, sessionID uuid.UUID) (*db.GetInterviewerLastMessageRow, error)
	GetMaxTurnNoBySessionID(ctx context.Context, sessionID uuid.UUID) (int64, error)
	CreateSessionTurnBySessionID(ctx context.Context, req *db.CreateInterviewTurnParams) error
	GetChatHistoryBySessionID(ctx context.Context, sessionID uuid.UUID) ([]db.GetChatHistoryBySessionIDRow, error)
}

type interviewTurnsRepository struct {
	log *log.Logger
	db  db.Store
}

func NewInterviewTurnsRepository(l *log.Logger, db db.Store) InterviewTurnsRepository {
	return &interviewTurnsRepository{
		log: l,
		db:  db,
	}
}

func (r *interviewTurnsRepository) GetInterviewerLastMessage(ctx context.Context, sessionID uuid.UUID) (*db.GetInterviewerLastMessageRow, error) {
	r.log.InfoWithID(ctx, "[Repository: GetInterviewerLastMessage] Called")

	turnRow, err := r.db.GetInterviewerLastMessage(ctx, sessionID)
	if err != nil {
		if err == sql.ErrNoRows {
			r.log.ErrorWithID(ctx, "[Repository: GetInterviewerLastMessage] Interviewer last message not found", err)
			return nil, app_error.New(err, app_error.ErrCodeInterviewTurnLastMessageNotFound)
		}
		r.log.ErrorWithID(ctx, "[Repository: GetInterviewerLastMessage] Error getting interviewer last message", err)
		return nil, app_error.HandleDatabaseError(err)
	}

	return &turnRow, nil
}

func (r *interviewTurnsRepository) GetMaxTurnNoBySessionID(ctx context.Context, sessionID uuid.UUID) (int64, error) {
	r.log.InfoWithID(ctx, "[Repository: GetMaxTurnNoBySessionID] Called")

	turnNo, err := r.db.GetMaxTurnNoBySessionID(ctx, sessionID)
	if err != nil {
		if err == sql.ErrNoRows {
			r.log.ErrorWithID(ctx, "[Repository: GetMaxTurnNoBySessionID] Max turn no by session ID not found", err)
			return 0, app_error.New(err, app_error.ErrCodeInterviewTurnsMaxTurnNoNotFound)
		}
		r.log.ErrorWithID(ctx, "[Repository: GetMaxTurnNoBySessionID] Error getting max turn no by session ID", err)
		return 0, app_error.HandleDatabaseError(err)
	}

	return turnNo.(int64), nil
}

func (r *interviewTurnsRepository) CreateSessionTurnBySessionID(ctx context.Context, req *db.CreateInterviewTurnParams) error {
	r.log.InfoWithID(ctx, "[Repository: CreateSessionTurnBySessionID] Called")

	if err := r.db.CreateInterviewTurn(ctx, *req); err != nil {
		r.log.ErrorWithID(ctx, "[Repository: CreateSessionTurnBySessionID] Error creating session turn by session ID", err)
		return app_error.HandleDatabaseError(err)
	}

	return nil
}

func (r *interviewTurnsRepository) GetChatHistoryBySessionID(ctx context.Context, sessionID uuid.UUID) ([]db.GetChatHistoryBySessionIDRow, error) {
	r.log.InfoWithID(ctx, "[Repository: GetChatHistoryBySessionID] Called")

	chatHistory, err := r.db.GetChatHistoryBySessionID(ctx, sessionID)
	if err != nil {
		r.log.ErrorWithID(ctx, "[Repository: GetChatHistoryBySessionID] Error getting chat history by session ID", err)
		return nil, app_error.HandleDatabaseError(err)
	}

	return chatHistory, nil
}
