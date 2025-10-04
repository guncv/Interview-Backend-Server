package services

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
	db "gitlab.com/interview-simulation/interview-backend-server/internal/db/sqlc"
	"gitlab.com/interview-simulation/interview-backend-server/internal/entities"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/database"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
	"gitlab.com/interview-simulation/interview-backend-server/internal/middleware"
	"gitlab.com/interview-simulation/interview-backend-server/internal/repositories"
	"gitlab.com/interview-simulation/interview-backend-server/internal/utils"
)

type IssueReportsService interface {
	CreateUserIssueReport(ctx context.Context, req *entities.CreateUserIssueReportReq) (*entities.UserIssueReport, error)
	ListIssueCategories(ctx context.Context) (*entities.ListIssueCategoriesResp, error)
	CreateAdminIssueCategory(ctx context.Context, req *entities.CreateAdminIssueCategoryReq) error
}

type issueReportsService struct {
	log                 *log.Logger
	issueReportsRepo    repositories.IssueReportsRepository
	issueCategoriesRepo repositories.IssueCategoriesRepository
	authContext         middleware.AuthContext
	redisClient         database.RedisClient
	generator           utils.Generator
}

func NewIssueReportsService(
	log *log.Logger,
	issueReportsRepo repositories.IssueReportsRepository,
	issueCategoriesRepo repositories.IssueCategoriesRepository,
	authContext middleware.AuthContext,
	redisClient database.RedisClient,
	generator utils.Generator,
) IssueReportsService {
	return &issueReportsService{
		log:                 log,
		issueReportsRepo:    issueReportsRepo,
		issueCategoriesRepo: issueCategoriesRepo,
		authContext:         authContext,
		redisClient:         redisClient,
		generator:           generator,
	}
}

func (s *issueReportsService) CreateUserIssueReport(ctx context.Context, req *entities.CreateUserIssueReportReq) (*entities.UserIssueReport, error) {

	authCtx, err := s.authContext.GetAuthContext(ctx)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: CreateUserIssueReport] Error getting auth context", err)
		return nil, err
	}

	userID, err := uuid.Parse(authCtx.Payload.UserID)
	if err != nil {
		err := app_error.New(err, app_error.ErrCodeGeneralInvalidUUID)
		s.log.ErrorWithID(ctx, "[Service: CreateUserIssueReport] Error parsing user ID", err)
		return nil, err
	}

	categoryID, err := uuid.Parse(req.CategoryID)
	if err != nil {
		err := app_error.New(err, app_error.ErrCodeGeneralInvalidUUID)
		s.log.ErrorWithID(ctx, "[Service: CreateUserIssueReport] Error parsing category ID", err)
		return nil, err
	}

	categoryName, err := s.issueCategoriesRepo.GetIssueCategoryIfExists(ctx, categoryID)
	if err != nil {
		err := app_error.New(err, app_error.ErrCodeGeneralInvalidUUID)
		s.log.ErrorWithID(ctx, "[Service: CreateUserIssueReport] Error checking issue category exists", err)
		return nil, err
	}

	reqDB := &db.CreateUserIssueReportParams{
		ID:          s.generator.GenerateUUID(ctx),
		UserID:      uuid.NullUUID{UUID: userID, Valid: true},
		Description: req.Description,
		Status:      constants.IssueReportStatusOpen,
		CategoryID:  categoryID,
		Priority:    constants.IssueReportPriorityNormal,
		CreatedAt:   time.Now(),
	}

	dbResp, err := s.issueReportsRepo.CreateUserIssueReport(ctx, reqDB)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: CreateUserIssueReport] Error creating user issue report", err)
		return nil, err
	}

	resp := &entities.UserIssueReport{
		ID:           dbResp.ID.String(),
		Description:  dbResp.Description,
		CategoryID:   dbResp.CategoryID.String(),
		CategoryName: categoryName.Name,
		IsEditable:   dbResp.Status == constants.IssueReportStatusOpen,
		Acknowledged: dbResp.Acknowledged,
		CommentCount: dbResp.CommentCount,
		CreatedAt:    dbResp.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    dbResp.UpdatedAt.Format(time.RFC3339),
	}

	return resp, nil
}

func (s *issueReportsService) ListIssueCategories(ctx context.Context) (*entities.ListIssueCategoriesResp, error) {

	var issueCategories []entities.IssueCategory
	redisData, err := s.redisClient.Get(ctx, constants.RedisPrefixIssueCategories)
	if err != nil {
		issueCategories, err = s.fetchIssueCategoriesFromDB(ctx)
		if err != nil {
			return nil, err
		}
	} else {
		err := json.Unmarshal([]byte(redisData), &issueCategories)
		if err != nil {
			s.log.WarnWithID(ctx, "[Service: ListIssueCategories] Error unmarshalling issue categories from redis, falling back to database", err)
			issueCategories, err = s.fetchIssueCategoriesFromDB(ctx)
			if err != nil {
				return nil, err
			}
		}
	}

	resp := &entities.ListIssueCategoriesResp{
		Data: issueCategories,
	}

	return resp, nil
}

func (s *issueReportsService) fetchIssueCategoriesFromDB(ctx context.Context) ([]entities.IssueCategory, error) {
	dbResp, err := s.issueCategoriesRepo.ListIssueCategories(ctx)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: ListIssueCategories] Error listing issue categories", err)
		return nil, err
	}

	issueCategories := make([]entities.IssueCategory, len(dbResp))
	for i, category := range dbResp {
		issueCategories[i] = entities.IssueCategory{
			ID:   category.ID.String(),
			Name: category.Name,
		}
	}

	if jsonBytes, err := json.Marshal(issueCategories); err == nil {
		go func() {
			cacheCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			_ = s.redisClient.Set(cacheCtx, database.RedisPayload{
				Key:   constants.RedisPrefixIssueCategories,
				Value: string(jsonBytes),
				TTL:   constants.RedisTTLIssueCategories,
			})
		}()
	}

	return issueCategories, nil
}

func (s *issueReportsService) CreateAdminIssueCategory(ctx context.Context, req *entities.CreateAdminIssueCategoryReq) error {

	authCtx, err := s.authContext.GetAuthContext(ctx)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: CreateAdminIssueCategory] Error getting auth context", err)
		return err
	}

	if authCtx.Payload.Role != constants.UserRoleAdmin {
		err = app_error.New(constants.ErrPermissionDenied, app_error.ErrCodeGeneralPermissionDenied)
		s.log.ErrorWithID(ctx, "[Service: CreateAdminIssueCategory] Unauthorized", err)
		return err
	}

	userID, err := uuid.Parse(authCtx.Payload.UserID)
	if err != nil {
		err := app_error.New(err, app_error.ErrCodeGeneralInvalidUUID)
		s.log.ErrorWithID(ctx, "[Service: CreateAdminIssueCategory] Error parsing user ID", err)
		return err
	}

	dbReq := &db.CreateAdminIssueCategoryParams{
		ID:        s.generator.GenerateUUID(ctx),
		Name:      req.Name,
		CreatedAt: time.Now(),
		CreatedBy: userID,
	}

	if err := s.issueCategoriesRepo.CreateAdminIssueCategory(ctx, dbReq); err != nil {
		s.log.ErrorWithID(ctx, "[Service: CreateAdminIssueCategory] Error creating admin issue category", err)
		return err
	}

	_ = s.redisClient.Delete(ctx, constants.RedisPrefixIssueCategories)

	return nil
}
