package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gitlab.com/interview-simulation/interview-backend-server/internal/entities"
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
// @Success 200 {object} entities.UserIssueReport "Create user issue report response"
// @Failure 400 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Invalid request parameters"
// @Failure 404 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Issue category not found"
// @Failure 401 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Unauthorized"
// @Failure 500 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Internal server error"
// @Router /issue-reports [post]
func (h *IssueReportsHandler) CreateUserIssueReport(c *gin.Context) {
	ctx := c.Request.Context()

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

	res, err := h.issueReportsService.CreateUserIssueReport(ctx, req)
	if err != nil {
		h.log.ErrorWithID(ctx, "[Handler: CreateUserIssueReport] Error creating user issue report", err)
		utils.RespondWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, res)
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
// @Param request body entities.CreateAdminIssueCategoryReq true "Create admin issue category request"
// @Security BearerAuth
// @Success 204
// @Failure 400 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Invalid request parameters"
// @Failure 401 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Unauthorized"
// @Failure 401 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Permission denied"
// @Failure 500 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Internal server error"
// @Router /issue-categories [post]
func (h *IssueReportsHandler) CreateAdminIssueCategory(c *gin.Context) {
	ctx := c.Request.Context()

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
