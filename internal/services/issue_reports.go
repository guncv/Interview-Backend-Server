package services

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
	db "gitlab.com/interview-simulation/interview-backend-server/internal/db/sqlc"
	"gitlab.com/interview-simulation/interview-backend-server/internal/entities"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
	"gitlab.com/interview-simulation/interview-backend-server/internal/middleware"
	"gitlab.com/interview-simulation/interview-backend-server/internal/repositories"
	"gitlab.com/interview-simulation/interview-backend-server/internal/utils"
)

type IssueReportsService interface {
	CreateUserIssueReport(ctx context.Context, req *entities.CreateUserIssueReportReq) (*entities.UserIssueReport, error)
	ListUserIssueReports(ctx context.Context) (*entities.ListUserIssueReportsResp, error)
	UpdateUserIssueReportByID(ctx context.Context, req *entities.UpdateUserIssueReportByIDReq, reportId string) (*entities.UserIssueReport, error)
	ListIssueCategories(ctx context.Context) (*entities.ListIssueCategoriesResp, error)
	CreateAdminIssueCategory(ctx context.Context, req *entities.CreateAdminIssueCategoryReq) error
}

type issueReportsService struct {
	log                 *log.Logger
	issueReportsRepo    repositories.IssueReportsRepository
	issueCategoriesRepo repositories.IssueCategoriesRepository
	authContext         middleware.AuthContext
	generator           utils.Generator
}

func NewIssueReportsService(
	log *log.Logger,
	issueReportsRepo repositories.IssueReportsRepository,
	issueCategoriesRepo repositories.IssueCategoriesRepository,
	authContext middleware.AuthContext,
	generator utils.Generator,
) IssueReportsService {
	return &issueReportsService{
		log:                 log,
		issueReportsRepo:    issueReportsRepo,
		issueCategoriesRepo: issueCategoriesRepo,
		authContext:         authContext,
		generator:           generator,
	}
}

func (s *issueReportsService) CreateUserIssueReport(ctx context.Context, req *entities.CreateUserIssueReportReq) (*entities.UserIssueReport, error) {
	s.log.InfoWithID(ctx, "[Service: CreateUserIssueReport] Called")

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

func (s *issueReportsService) ListUserIssueReports(ctx context.Context) (*entities.ListUserIssueReportsResp, error) {
	s.log.InfoWithID(ctx, "[Service: ListUserIssueReports] Called")

	authCtx, err := s.authContext.GetAuthContext(ctx)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: ListUserIssueReports] Error getting auth context", err)
		return nil, err
	}

	userID, err := uuid.Parse(authCtx.Payload.UserID)
	if err != nil {
		err := app_error.New(err, app_error.ErrCodeGeneralInvalidUUID)
		s.log.ErrorWithID(ctx, "[Service: ListUserIssueReports] Error parsing user ID", err)
		return nil, err
	}

	dbResp, err := s.issueReportsRepo.ListUserIssueReports(ctx, userID)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: ListUserIssueReports] Error listing user issue reports", err)
		return nil, err
	}

	userIssueReports := make([]entities.UserIssueReport, len(dbResp))
	for i, report := range dbResp {
		userIssueReports[i] = entities.UserIssueReport{
			ID:           report.ID.String(),
			Description:  report.Description,
			CategoryID:   report.CategoryID.String(),
			CategoryName: report.CategoryName.String,
			IsEditable:   report.Status == constants.IssueReportStatusOpen,
			Acknowledged: report.Acknowledged,
			CommentCount: report.CommentCount,
			CreatedAt:    report.CreatedAt.Format(time.RFC3339),
			UpdatedAt:    report.UpdatedAt.Format(time.RFC3339),
		}
	}

	resp := &entities.ListUserIssueReportsResp{
		Data: userIssueReports,
	}

	return resp, nil
}

func (s *issueReportsService) UpdateUserIssueReportByID(ctx context.Context, req *entities.UpdateUserIssueReportByIDReq, reportId string) (*entities.UserIssueReport, error) {
	s.log.InfoWithID(ctx, "[Service: UpdateUserIssueReportByID] Called")

	authCtx, err := s.authContext.GetAuthContext(ctx)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: UpdateUserIssueReportByID] Error getting auth context", err)
		return nil, err
	}

	userID, err := uuid.Parse(authCtx.Payload.UserID)
	if err != nil {
		err := app_error.New(err, app_error.ErrCodeGeneralInvalidUUID)
		s.log.ErrorWithID(ctx, "[Service: UpdateUserIssueReportByID] Error parsing user ID", err)
		return nil, err
	}

	issueReportID, err := uuid.Parse(reportId)
	if err != nil {
		err := app_error.New(err, app_error.ErrCodeGeneralInvalidUUID)
		s.log.ErrorWithID(ctx, "[Service: UpdateUserIssueReportByID] Error parsing issue report ID", err)
		return nil, err
	}

	categoryID, err := uuid.Parse(req.CategoryID)
	if err != nil {
		err := app_error.New(err, app_error.ErrCodeGeneralInvalidUUID)
		s.log.ErrorWithID(ctx, "[Service: UpdateUserIssueReportByID] Error parsing category ID", err)
		return nil, err
	}

	categoryName, err := s.issueCategoriesRepo.GetIssueCategoryIfExists(ctx, categoryID)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: UpdateUserIssueReportByID] Error checking issue category exists", err)
		return nil, err
	}

	issueReportUserIDAndStatus, err := s.issueReportsRepo.GetUserIssueReportUserIDAndStatusByID(ctx, issueReportID)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: UpdateUserIssueReportByID] Error getting user issue report user ID and status", err)
		return nil, err
	}

	if issueReportUserIDAndStatus.UserID.UUID != userID {
		err = app_error.New(constants.ErrIssueReportUnauthorized, app_error.ErrCodeIssueReportUnauthorized)
		s.log.ErrorWithID(ctx, "[Service: UpdateUserIssueReportByID] Unauthorized", err)
		return nil, err
	}

	if issueReportUserIDAndStatus.Status != constants.IssueReportStatusOpen {
		err = app_error.New(constants.ErrIssueReportNotOpen, app_error.ErrCodeIssueReportNotOpen)
		s.log.ErrorWithID(ctx, "[Service: UpdateUserIssueReportByID] Issue report not open", err)
		return nil, err
	}

	reqDB := &db.UpdateUserIssueReportByIDParams{
		ID:          issueReportID,
		Description: req.Description,
		CategoryID:  categoryID,
		UpdatedAt:   time.Now(),
	}

	dbResp, err := s.issueReportsRepo.UpdateUserIssueReportByID(ctx, reqDB)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: UpdateUserIssueReportByID] Error updating user issue report", err)
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
	s.log.InfoWithID(ctx, "[Service: ListIssueCategories] Called")

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

	resp := &entities.ListIssueCategoriesResp{
		Data: issueCategories,
	}

	return resp, nil
}

func (s *issueReportsService) CreateAdminIssueCategory(ctx context.Context, req *entities.CreateAdminIssueCategoryReq) error {
	s.log.InfoWithID(ctx, "[Service: CreateAdminIssueCategory] Called")

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

	return nil
}
