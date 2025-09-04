package repositories

import (
	"context"

	"github.com/google/uuid"
	db "gitlab.com/interview-simulation/interview-backend-server/internal/db/sqlc"
	app_error "gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	log "gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
)

type JobRequirementRepository interface {
	CreateJobRequirement(ctx context.Context, req *db.CreateJobRequirementParams) error
	DeleteJobRequirement(ctx context.Context, id uuid.UUID) error
}

type jobRequirementRepository struct {
	log *log.Logger
	db  db.Queries
}

func NewJobRequirementRepository(
	log *log.Logger,
	db db.Queries,
) JobRequirementRepository {
	return &jobRequirementRepository{
		log: log,
		db:  db,
	}
}

func (r *jobRequirementRepository) CreateJobRequirement(ctx context.Context, req *db.CreateJobRequirementParams) error {
	r.log.InfoWithID(ctx, "[Repository: CreateJobRequirement] Called")

	if err := r.db.CreateJobRequirement(ctx, *req); err != nil {
		r.log.ErrorWithID(ctx, "[Repository: CreateJobRequirement] Error creating job requirement", err)
		return app_error.HandleDatabaseError(err)
	}

	return nil
}

func (r *jobRequirementRepository) DeleteJobRequirement(ctx context.Context, id uuid.UUID) error {
	r.log.InfoWithID(ctx, "[Repository: DeleteJobRequirement] Called")

	if err := r.db.DeleteJobRequirement(ctx, id); err != nil {
		r.log.ErrorWithID(ctx, "[Repository: DeleteJobRequirement] Error deleting job requirement", err)
		return app_error.HandleDatabaseError(err)
	}

	return nil
}
