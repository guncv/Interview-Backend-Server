package repositories

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/google/uuid"
	"gitlab.com/interview-simulation/interview-backend-server/internal/config"
	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
	db "gitlab.com/interview-simulation/interview-backend-server/internal/db/sqlc"
	app_error "gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
)

type ResumeReposity interface {
	CreateResume(ctx context.Context, req *db.CreateResumeParams) error
	ListResumeByUserIDFirstPage(ctx context.Context, userID uuid.UUID) ([]db.Resumes, error)
	ListResumeByUserIDPaginated(ctx context.Context, req *db.ListResumeByUserIDPaginatedParams) ([]db.Resumes, error)
	CheckIsDefaultResumeExistsByUserID(ctx context.Context, userID uuid.UUID) (bool, error)
	GetResumeByID(ctx context.Context, id uuid.UUID) (*db.Resumes, error)
	GetDefaultResumeByUserID(ctx context.Context, userID uuid.UUID) (*db.Resumes, error)
	SwitchDefaultResume(ctx context.Context, oldID, newID uuid.UUID) error
	ExtractResumeJsonForRAG(ctx context.Context, req *ExtractResumeJsonForRAGReq) error
	ListAllResumesFileNameByUserID(ctx context.Context, userID uuid.UUID) ([]string, error)
}

type resumeRepository struct {
	log *log.Logger
	db  db.Store
	cfg *config.Config
}

func NewResumeRepository(l *log.Logger, db db.Store, cfg *config.Config) ResumeReposity {
	return &resumeRepository{
		log: l,
		db:  db,
		cfg: cfg,
	}
}

func (r *resumeRepository) CreateResume(ctx context.Context, req *db.CreateResumeParams) error {
	r.log.InfoWithID(ctx, "[Repository: CreateResume] Called")

	if err := r.db.CreateResume(ctx, *req); err != nil {
		r.log.ErrorWithID(ctx, "[Repository: CreateResume] Error creating resume", err)
		return app_error.HandleDatabaseError(err)
	}

	return nil
}

func (r *resumeRepository) ListResumeByUserIDFirstPage(ctx context.Context, userID uuid.UUID) ([]db.Resumes, error) {
	r.log.InfoWithID(ctx, "[Repository: ListResumeByUserIDFirstPage] Called")

	resumes, err := r.db.ListResumeByUserIDFirstPage(ctx, userID)
	if err != nil {
		r.log.ErrorWithID(ctx, "[Repository: ListResumeByUserIDFirstPage] Error getting list resume", err)
		return nil, app_error.HandleDatabaseError(err)
	}

	return resumes, nil
}

func (r *resumeRepository) ListResumeByUserIDPaginated(ctx context.Context, req *db.ListResumeByUserIDPaginatedParams) ([]db.Resumes, error) {
	r.log.InfoWithID(ctx, "[Repository: ListResumeByUserIDPaginated] Called")

	resumes, err := r.db.ListResumeByUserIDPaginated(ctx, *req)
	if err != nil {
		r.log.ErrorWithID(ctx, "[Repository: ListResumeByUserIDPaginated] Error getting list resume", err)
		return nil, app_error.HandleDatabaseError(err)
	}

	return resumes, nil
}

func (r *resumeRepository) CheckIsDefaultResumeExistsByUserID(ctx context.Context, userID uuid.UUID) (bool, error) {
	r.log.InfoWithID(ctx, "[Repository: CheckIsDefaultResumeExistsByUserID] Called")

	exists, err := r.db.CheckIsDefaultResumeExistsByUserID(ctx, userID)
	if err != nil {
		r.log.ErrorWithID(ctx, "[Repository: CheckIsDefaultResumeExistsByUserID] Error checking if default resume exists", err)
		return false, app_error.HandleDatabaseError(err)
	}

	return exists, nil
}

func (r *resumeRepository) GetResumeByID(ctx context.Context, id uuid.UUID) (*db.Resumes, error) {
	r.log.InfoWithID(ctx, "[Repository: GetResumeByID] Called")

	resume, err := r.db.GetResumeByID(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			r.log.ErrorWithID(ctx, "[Repository: GetResumeByID] Resume not found", err)
			return nil, app_error.New(err, app_error.ErrCodeResumeNotFound)
		}
		r.log.ErrorWithID(ctx, "[Repository: GetResumeByID] Error getting resume", err)
		return nil, app_error.HandleDatabaseError(err)
	}

	return &resume, nil
}

func (r *resumeRepository) GetDefaultResumeByUserID(ctx context.Context, userID uuid.UUID) (*db.Resumes, error) {
	r.log.InfoWithID(ctx, "[Repository: GetDefaultResumeByUserID] Called")

	defaultResume, err := r.db.GetDefaultResumeByUserID(ctx, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			r.log.ErrorWithID(ctx, "[Repository: GetDefaultResumeByUserID] Default resume not found", err)
			return nil, app_error.New(err, app_error.ErrCodeResumeNotFound)
		}
		r.log.ErrorWithID(ctx, "[Repository: GetDefaultResumeByUserID] Error getting default resume", err)
		return nil, app_error.HandleDatabaseError(err)
	}

	return &defaultResume, nil
}

func (r *resumeRepository) SwitchDefaultResume(ctx context.Context, oldID, newID uuid.UUID) error {
	r.log.InfoWithID(ctx, "[Repository: SwitchDefaultResume] Called")

	err := r.db.ExecTx(ctx, func(q *db.Queries) error {
		if err := q.UnsetDefaultResume(ctx, oldID); err != nil {
			r.log.ErrorWithID(ctx, "[Repository: SwitchDefaultResume] Error switching default resume", err)
			return app_error.HandleDatabaseError(err)
		}

		if err := q.SetDefaultResume(ctx, newID); err != nil {
			r.log.ErrorWithID(ctx, "[Repository: SwitchDefaultResume] Error switching default resume", err)
			return app_error.HandleDatabaseError(err)
		}

		return nil
	})

	if err != nil {
		r.log.ErrorWithID(ctx, "[Repository: SwitchDefaultResume] Error switching default resume", err)
		return app_error.HandleDatabaseError(err)
	}

	return nil
}

func (r *resumeRepository) ExtractResumeJsonForRAG(ctx context.Context, req *ExtractResumeJsonForRAGReq) error {
	r.log.InfoWithID(ctx, "[Repository: ExtractResumeJsonForRAG] Called")

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	if err := writer.WriteField("session_id", req.SessionID); err != nil {
		r.log.ErrorWithID(ctx, "[Repository: ExtractResumeJsonForRAG] Failed to write session_id field", err)
		return err
	}
	if err := writer.WriteField("user_id", req.UserID); err != nil {
		r.log.ErrorWithID(ctx, "[Repository: ExtractResumeJsonForRAG] Failed to write user_id field", err)
		return err
	}
	if err := writer.WriteField("resume_id", req.ResumeID); err != nil {
		r.log.ErrorWithID(ctx, "[Repository: ExtractResumeJsonForRAG] Failed to write resume_id field", err)
		return err
	}

	if req.ResumeFile != nil {
		part, err := writer.CreateFormFile("resume_file", req.ResumeFile.Filename)
		if err != nil {
			r.log.ErrorWithID(ctx, "[Repository: ExtractResumeJsonForRAG] Failed to create form file", err)
			return err
		}

		file, err := req.ResumeFile.Open()
		if err != nil {
			r.log.ErrorWithID(ctx, "[Repository: ExtractResumeJsonForRAG] Failed to open uploaded file", err)
			return err
		}
		defer file.Close()

		if _, err := io.Copy(part, file); err != nil {
			r.log.ErrorWithID(ctx, "[Repository: ExtractResumeJsonForRAG] Failed to write resume file data", err)
			return err
		}
	}

	if err := writer.Close(); err != nil {
		r.log.ErrorWithID(ctx, "[Repository: ExtractResumeJsonForRAG] Failed to close multipart writer", err)
		return err
	}

	endpoint := r.cfg.InterviewSessionConfig.InterviewAgentURL + constants.PathExtractResumeRAGAgent
	httpReq, err := http.NewRequest("POST", endpoint, &body)
	if err != nil {
		r.log.ErrorWithID(ctx, "[Repository: ExtractResumeJsonForRAG] Failed to create HTTP request", err)
		return err
	}

	httpReq.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{Timeout: constants.TimeoutHTTP}
	resp, err := client.Do(httpReq)
	if err != nil {
		r.log.ErrorWithID(ctx, "[Repository: ExtractResumeJsonForRAG] HTTP request failed", err)
		return err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		r.log.ErrorWithID(ctx, "[Repository: ExtractResumeJsonForRAG] Failed to read response body", err)
		return err
	}

	r.log.InfoWithID(ctx, fmt.Sprintf("[Repository: ExtractResumeJsonForRAG] Response: %s | Body: %s", resp.Status, string(bodyBytes)))

	if resp.StatusCode != http.StatusNoContent {
		err := fmt.Errorf("interview agent returned status: %s | body: %s", resp.Status, string(bodyBytes))
		r.log.ErrorWithID(ctx, "[Repository: ExtractResumeJsonForRAG] HTTP request failed", err)
		return err
	}

	return nil
}

func (r *resumeRepository) ListAllResumesFileNameByUserID(ctx context.Context, userID uuid.UUID) ([]string, error) {
	r.log.InfoWithID(ctx, "[Repository: ListAllResumesFileNameByUserID] Called")

	resumes, err := r.db.ListAllResumesFileNameByUserID(ctx, userID)
	if err != nil {
		r.log.ErrorWithID(ctx, "[Repository: ListAllResumesFileNameByUserID] Error getting list resume", err)
		return nil, app_error.HandleDatabaseError(err)
	}

	return resumes, nil
}
