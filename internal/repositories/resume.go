package repositories

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/google/uuid"
	"gitlab.com/interview-simulation/interview-backend-server/internal/config"
	db "gitlab.com/interview-simulation/interview-backend-server/internal/db/sqlc"
	app_error "gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
)

type ResumeReposity interface {
	CreateResume(ctx context.Context, req *db.CreateResumeParams) error
	ListResumeByUserID(ctx context.Context, req *db.ListResumeByUserIDParams) ([]db.Resumes, error)
	CheckIsDefaultResumeExistsByUserID(ctx context.Context, userID uuid.UUID) (bool, error)
	GetDefaultResumeByUserID(ctx context.Context, userID uuid.UUID) (db.Resumes, error)
	SwitchDefaultResume(ctx context.Context, oldID, newID uuid.UUID) error
	GetResumeJsonWithSummaryData(ctx context.Context, req *GetResumeJsonWithSummaryDataReq) (*GetResumeJsonWithSummaryDataResponse, error)
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

func (r *resumeRepository) ListResumeByUserID(ctx context.Context, req *db.ListResumeByUserIDParams) ([]db.Resumes, error) {
	r.log.InfoWithID(ctx, "[Repository: ListResumeByUserID] Called")

	resumes, err := r.db.ListResumeByUserID(ctx, *req)
	if err != nil {
		r.log.ErrorWithID(ctx, "[Repository: ListResumeByUserID] Error getting list resume", err)
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

func (r *resumeRepository) GetDefaultResumeByUserID(ctx context.Context, userID uuid.UUID) (db.Resumes, error) {
	r.log.InfoWithID(ctx, "[Repository: GetDefaultResumeByUserID] Called")

	defaultResume, err := r.db.GetDefaultResumeByUserID(ctx, userID)
	if err != nil {
		r.log.ErrorWithID(ctx, "[Repository: GetDefaultResumeByUserID] Error getting default resume", err)
		return db.Resumes{}, app_error.HandleDatabaseError(err)
	}

	return defaultResume, nil
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

func (r *resumeRepository) GetResumeJsonWithSummaryData(ctx context.Context, req *GetResumeJsonWithSummaryDataReq) (*GetResumeJsonWithSummaryDataResponse, error) {
	r.log.InfoWithID(ctx, "[Repository: GetResumeJsonWithSummaryData] Called")

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	if err := writer.WriteField("session_id", req.SessionID.String()); err != nil {
		r.log.ErrorWithID(ctx, "[Repository: GetResumeJsonWithSummaryData] Failed to write session_id field", err)
		return nil, err
	}
	if err := writer.WriteField("position", req.Position); err != nil {
		r.log.ErrorWithID(ctx, "[Repository: GetResumeJsonWithSummaryData] Failed to write position field", err)
		return nil, err
	}
	if err := writer.WriteField("company", req.Company); err != nil {
		r.log.ErrorWithID(ctx, "[Repository: GetResumeJsonWithSummaryData] Failed to write company field", err)
		return nil, err
	}
	if err := writer.WriteField("work_type", req.WorkType); err != nil {
		r.log.ErrorWithID(ctx, "[Repository: GetResumeJsonWithSummaryData] Failed to write work_type field", err)
		return nil, err
	}
	if err := writer.WriteField("job_requirements", req.JobRequirements); err != nil {
		r.log.ErrorWithID(ctx, "[Repository: GetResumeJsonWithSummaryData] Failed to write job_requirements field", err)
		return nil, err
	}
	if err := writer.WriteField("interview_type", req.InterviewType); err != nil {
		r.log.ErrorWithID(ctx, "[Repository: GetResumeJsonWithSummaryData] Failed to write interview_type field", err)
		return nil, err
	}
	if err := writer.WriteField("language", req.Language); err != nil {
		r.log.ErrorWithID(ctx, "[Repository: GetResumeJsonWithSummaryData] Failed to write language field", err)
		return nil, err
	}

	if req.ResumeFile != nil {
		part, err := writer.CreateFormFile("resume_file", req.ResumeFile.Filename)
		if err != nil {
			r.log.ErrorWithID(ctx, "[Repository: GetResumeJsonWithSummaryData] Failed to create form file", err)
			return nil, err
		}

		file, err := req.ResumeFile.Open()
		if err != nil {
			r.log.ErrorWithID(ctx, "[Repository: GetResumeJsonWithSummaryData] Failed to open uploaded file", err)
			return nil, err
		}
		defer file.Close()

		if _, err := io.Copy(part, file); err != nil {
			r.log.ErrorWithID(ctx, "[Repository: GetResumeJsonWithSummaryData] Failed to write resume file data", err)
			return nil, err
		}
	}

	if err := writer.Close(); err != nil {
		r.log.ErrorWithID(ctx, "[Repository: GetResumeJsonWithSummaryData] Failed to close multipart writer", err)
		return nil, err
	}

	endpoint := r.cfg.InterviewAgentConfig.InterviewAgentURL + "/api/v1/interview/requirements"
	httpReq, err := http.NewRequest("POST", endpoint, &body)
	if err != nil {
		r.log.ErrorWithID(ctx, "[Repository: GetResumeJsonWithSummaryData] Failed to create HTTP request", err)
		return nil, err
	}

	httpReq.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		r.log.ErrorWithID(ctx, "[Repository: GetResumeJsonWithSummaryData] HTTP request failed", err)
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		r.log.ErrorWithID(ctx, "[Repository: GetResumeJsonWithSummaryData] Failed to read response body", err)
		return nil, err
	}

	r.log.InfoWithID(ctx, fmt.Sprintf("[Repository: GetResumeJsonWithSummaryData] Response: %s | Body: %s", resp.Status, string(bodyBytes)))

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("interview agent returned status: %s | body: %s", resp.Status, string(bodyBytes))
		r.log.ErrorWithID(ctx, "[Repository: GetResumeJsonWithSummaryData] HTTP request failed", err)
		return nil, err
	}

	var response GetResumeJsonWithSummaryDataResponse
	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		r.log.ErrorWithID(ctx, "[Repository: GetResumeJsonWithSummaryData] Failed to unmarshal response", err)
		return nil, err
	}

	return &response, nil
}
