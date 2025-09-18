package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
	"gitlab.com/interview-simulation/interview-backend-server/internal/entities"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
	"gitlab.com/interview-simulation/interview-backend-server/internal/middleware"
	"gitlab.com/interview-simulation/interview-backend-server/internal/services"
	"gitlab.com/interview-simulation/interview-backend-server/internal/utils"
)

type IssueReportsHandler struct {
	log                 *log.Logger
	issueReportsService services.IssueReportsService
	authContext         middleware.AuthContext
	validator           utils.Validator
}

func NewIssueReportsHandler(
	log *log.Logger,
	issueReportsService services.IssueReportsService,
	authContext middleware.AuthContext,
	validator utils.Validator,
) *IssueReportsHandler {
	return &IssueReportsHandler{
		log:                 log,
		issueReportsService: issueReportsService,
		authContext:         authContext,
		validator:           validator,
	}
}

// CreateUserIssueReport godoc
// @Summary Create user issue report
// @Description Create a new user issue report
// @Tags Issue Reports
// @Accept json
// @Produce json
// @Param request body entities.CreateUserIssueReportReq true "Create user issue report request"
// @Security BearerAuth
// @Success 204
// @Failure 400 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Invalid request parameters"
// @Failure 404 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Issue category not found"
// @Failure 401 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Unauthorized"
// @Failure 500 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Internal server error"
// @Router /issue-reports [post]
func (h *IssueReportsHandler) CreateUserIssueReport(c *gin.Context) {
	ctx := c.Request.Context()
	h.log.InfoWithID(ctx, "[Handler: CreateUserIssueReport] Called")

	req := &entities.CreateUserIssueReportReq{}
	if err := h.validator.ValidateAndBind(c, req, "CreateUserIssueReport"); err != nil {
		h.log.ErrorWithID(ctx, "[Handler: CreateUserIssueReport] Error binding JSON", err)
		utils.RespondWithError(c, err)
		return
	}

	ctx, err := h.authContext.ExtractAuthContext(c)
	if err != nil {
		h.log.ErrorWithID(ctx, "[Handler: CreateUserIssueReport] Error getting auth context", err)
		utils.RespondWithError(c, err)
		return
	}

	err = h.issueReportsService.CreateUserIssueReport(ctx, req)
	if err != nil {
		h.log.ErrorWithID(ctx, "[Handler: CreateUserIssueReport] Error creating user issue report", err)
		utils.RespondWithError(c, err)
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// ListUserIssueReports godoc
// @Summary List user issue reports
// @Description Get a list of all user issue reports for the authenticated user
// @Tags Issue Reports
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} entities.ListUserIssueReportsResp "List user issue reports response"
// @Failure 400 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Invalid request parameters"
// @Failure 401 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Unauthorized"
// @Failure 500 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Internal server error"
// @Router /issue-reports [get]
func (h *IssueReportsHandler) ListUserIssueReports(c *gin.Context) {
	ctx := c.Request.Context()
	h.log.InfoWithID(ctx, "[Handler: ListUserIssueReports] Called")

	ctx, err := h.authContext.ExtractAuthContext(c)
	if err != nil {
		h.log.ErrorWithID(ctx, "[Handler: ListUserIssueReports] Error getting auth context", err)
		utils.RespondWithError(c, err)
		return
	}

	res, err := h.issueReportsService.ListUserIssueReports(ctx)
	if err != nil {
		h.log.ErrorWithID(ctx, "[Handler: ListUserIssueReports] Error listing user issue reports", err)
		utils.RespondWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, res)
}

// UpdateUserIssueReportByID godoc
// @Summary Update user issue report by ID
// @Description Update a user issue report by ID
// @Tags Issue Reports
// @Accept json
// @Produce json
// @Param request body entities.UpdateUserIssueReportByIDReq true "Update user issue report request"
// @Security BearerAuth
// @Success 204
// @Failure 400 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Invalid request parameters"
// @Failure 404 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Issue category not found"
// @Failure 404 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Issue report not found"
// @Failure 404 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "User is not the owner of the issue report"
// @Failure 401 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Unauthorized"
// @Failure 500 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Internal server error"
// @Router /issue-reports/{issue_report_id} [patch]
func (h *IssueReportsHandler) UpdateUserIssueReportByID(c *gin.Context) {
	ctx := c.Request.Context()
	h.log.InfoWithID(ctx, "[Handler: UpdateUserIssueReportByID] Called")

	reportId := c.Param("issue_report_id")
	if reportId == "" {
		h.log.ErrorWithID(ctx, "[Handler: UpdateUserIssueReportByID] Issue report ID is required")
		utils.RespondWithError(c, app_error.New(constants.ErrIssueReportIDRequired, app_error.ErrCodeIssueReportIDRequired))
		return
	}

	if err := h.validator.GetValidate().Var(reportId, "required,uuid"); err != nil {
		h.log.ErrorWithID(ctx, "[Handler: UpdateUserIssueReportByID] Invalid issue report ID", err)
		utils.RespondWithError(c, app_error.New(err, app_error.ErrCodeIssueReportIDRequired))
		return
	}

	req := &entities.UpdateUserIssueReportByIDReq{}
	if err := h.validator.ValidateAndBind(c, req, "UpdateUserIssueReportByID"); err != nil {
		h.log.ErrorWithID(ctx, "[Handler: UpdateUserIssueReportByID] Error binding JSON", err)
		utils.RespondWithError(c, err)
		return
	}

	ctx, err := h.authContext.ExtractAuthContext(c)
	if err != nil {
		h.log.ErrorWithID(ctx, "[Handler: UpdateUserIssueReportByID] Error getting auth context", err)
		utils.RespondWithError(c, err)
		return
	}

	err = h.issueReportsService.UpdateUserIssueReportByID(ctx, req, reportId)
	if err != nil {
		h.log.ErrorWithID(ctx, "[Handler: UpdateUserIssueReportByID] Error updating user issue report", err)
		utils.RespondWithError(c, err)
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// ListIssueCategories godoc
// @Summary List issue categories
// @Description Get a list of all issue categories
// @Tags Issue Reports
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} entities.ListIssueCategoriesResp "List issue categories response"
// @Failure 401 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Unauthorized"
// @Failure 500 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Internal server error"
// @Router /issue-categories [get]
func (h *IssueReportsHandler) ListIssueCategories(c *gin.Context) {
	ctx := c.Request.Context()
	h.log.InfoWithID(ctx, "[Handler: ListIssueCategories] Called")

	res, err := h.issueReportsService.ListIssueCategories(ctx)
	if err != nil {
		h.log.ErrorWithID(ctx, "[Handler: ListIssueCategories] Error listing issue categories", err)
		utils.RespondWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, res)
}

// CreateAdminIssueCategory godoc
// @Summary Create admin issue category
// @Description Create a new admin issue category
// @Tags Issue Reports
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 204
// @Failure 400 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Invalid request parameters"
// @Failure 401 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Unauthorized"
// @Failure 401 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Permission denied"
// @Failure 500 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Internal server error"
// @Router /issue-categories [post]
func (h *IssueReportsHandler) CreateAdminIssueCategory(c *gin.Context) {
	ctx := c.Request.Context()
	h.log.InfoWithID(ctx, "[Handler: CreateAdminIssueCategory] Called")

	req := &entities.CreateAdminIssueCategoryReq{}
	if err := h.validator.ValidateAndBind(c, req, "CreateAdminIssueCategory"); err != nil {
		h.log.ErrorWithID(ctx, "[Handler: CreateAdminIssueCategory] Error binding JSON", err)
		utils.RespondWithError(c, err)
		return
	}

	ctx, err := h.authContext.ExtractAuthContext(c)
	if err != nil {
		h.log.ErrorWithID(ctx, "[Handler: CreateAdminIssueCategory] Error getting auth context", err)
		utils.RespondWithError(c, err)
		return
	}

	err = h.issueReportsService.CreateAdminIssueCategory(ctx, req)
	if err != nil {
		h.log.ErrorWithID(ctx, "[Handler: CreateAdminIssueCategory] Error creating admin issue category", err)
		utils.RespondWithError(c, err)
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
