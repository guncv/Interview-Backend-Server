package handlers

import (
	"net/http"

	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"gitlab.com/interview-simulation/interview-backend-server/internal/entities"
	app_error "gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
	"gitlab.com/interview-simulation/interview-backend-server/internal/middleware"
	"gitlab.com/interview-simulation/interview-backend-server/internal/services"
	"gitlab.com/interview-simulation/interview-backend-server/internal/utils"
)

type ResumeHandler struct {
	resumeService services.ResumeService
	log           *log.Logger
	authContext   middleware.AuthContext
	validator     utils.Validator
}

func NewResumeHandler(
	resumeService services.ResumeService,
	log *log.Logger,
	authContext middleware.AuthContext,
	validator utils.Validator) *ResumeHandler {
	return &ResumeHandler{
		resumeService: resumeService,
		log:           log,
		authContext:   authContext,
		validator:     validator,
	}
}

// ListResume godoc
// @Summary List user resumes
// @Description Get a list of all resumes for the authenticated user
// @Tags Resumes
// @Accept json
// @Produce json
// @Param updated_at query string false "Filter by updated date (RFC3339 format)"
// @Security BearerAuth
// @Success 200 {object} entities.ListResumeResponse
// @Failure 400 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Invalid request parameters"
// @Failure 401 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Unauthorized"
// @Failure 500 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Internal server error"
// @Router /resumes/list [get]
func (h *ResumeHandler) ListResume(c *gin.Context) {
	ctx := c.Request.Context()

	updatedAtStr := c.Query("updated_at")
	var req entities.ListResumeRequest

	if updatedAtStr != "" {
		updatedAt, err := time.Parse(time.RFC3339, updatedAtStr)
		if err != nil {
			h.log.ErrorWithID(ctx, "[Handler: ListResume] Invalid updated_at format", err)
			utils.RespondWithError(c, app_error.New(err, app_error.ErrCodeResumeInvalidRequest))
			return
		}
		req.UpdatedAt = &updatedAt
	}

	ctx, err := h.authContext.ExtractAuthContext(c)
	if err != nil {
		h.log.ErrorWithID(ctx, "[Handler: ListResume] Error getting auth context", err)
		utils.RespondWithError(c, err)
		return
	}

	resp, err := h.resumeService.ListResume(ctx, &req)
	if err != nil {
		h.log.ErrorWithID(ctx, "[Handler: ListResume] Error getting resume list", err)
		utils.RespondWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// SwitchDefaultResume godoc
// @Summary Switch default resume
// @Description Set a specific resume as the default resume for the user
// @Tags Resumes
// @Accept json
// @Produce json
// @Param request body entities.SwitchDefaultResumeRequest true "Resume switch request"
// @Security BearerAuth
// @Success 200 "Default resume switched successfully"
// @Failure 400 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Validation error or business logic error"
// @Failure 401 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Unauthorized"
// @Failure 500 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Internal server error"
// @Router /resumes/switch-default [post]
func (h *ResumeHandler) SwitchDefaultResume(c *gin.Context) {
	var req entities.SwitchDefaultResumeRequest
	if err := h.validator.ValidateAndBind(c, &req, "SwitchDefaultResume"); err != nil {
		utils.RespondWithError(c, err)
		return
	}

	ctx, err := h.authContext.ExtractAuthContext(c)
	if err != nil {
		h.log.ErrorWithID(ctx, "[Handler: SwitchDefaultResume] Error getting auth context", err)
		utils.RespondWithError(c, err)
		return
	}

	err = h.resumeService.SwitchDefaultResume(ctx, &req)
	if err != nil {
		h.log.ErrorWithID(ctx, "[Handler: SwitchDefaultResume] Error switching default resume", err)
		utils.RespondWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, nil)
}

// GetResumeByID godoc
// @Summary Get resume by ID
// @Description Retrieve a specific resume by its ID
// @Tags Resumes
// @Accept json
// @Produce json
// @Param id path string true "Resume ID (UUID)"
// @Security BearerAuth
// @Success 200 {object} entities.GetResumeByIDResponse
// @Failure 400 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Invalid resume ID"
// @Failure 401 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Unauthorized"
// @Failure 404 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Resume not found"
// @Failure 500 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Internal server error"
// @Router /resumes/{id} [get]
func (h *ResumeHandler) GetResumeByID(c *gin.Context) {
	ctx := c.Request.Context()

	resumeId := c.Param("id")
	if resumeId == "" {
		h.log.ErrorWithID(ctx, "[Handler: GetResumeByID] Resume ID is required")
		utils.RespondWithError(c, app_error.New(errors.New("resume ID is required"), app_error.ErrCodeResumeInvalidRequest))
		return
	}

	if err := h.validator.GetValidate().Var(resumeId, "required,uuid"); err != nil {
		h.log.ErrorWithID(ctx, "[Handler: GetResumeByID] Invalid resume ID", err)
		utils.RespondWithError(c, app_error.New(err, app_error.ErrCodeResumeInvalidID))
		return
	}

	resp, err := h.resumeService.GetResumeByID(ctx, &entities.GetResumeByIDRequest{
		ResumeID: resumeId,
	})
	if err != nil {
		h.log.ErrorWithID(ctx, "[Handler: GetResumeByID] Error getting resume", err)
		utils.RespondWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// DownloadResumeByResumeId godoc
// @Summary Download resume by resume ID
// @Description Download a specific resume by its resume ID
// @Tags Resumes
// @Accept json
// @Produce json
// @Param resume_id path string true "Resume ID (UUID)"
// @Security BearerAuth
// @Success 200 {object} entities.DownloadResumeByResumeIdResp
// @Failure 400 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Invalid resume ID"
// @Failure 401 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Unauthorized"
// @Failure 404 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Session not found"
// @Failure 500 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Internal server error"
// @Router /resumes/download/{resume_id} [get]
func (h *ResumeHandler) DownloadResumeByResumeId(c *gin.Context) {
	ctx := c.Request.Context()

	resumeId := c.Param("resume_id")
	if resumeId == "" {
		h.log.ErrorWithID(ctx, "[Handler: DownloadResumeByResumeId] Resume ID is required")
		utils.RespondWithError(c, app_error.New(errors.New("resume ID is required"), app_error.ErrCodeResumeInvalidRequest))
		return
	}

	if err := h.validator.GetValidate().Var(resumeId, "required,uuid"); err != nil {
		h.log.ErrorWithID(ctx, "[Handler: DownloadResumeByResumeId] Invalid resume ID", err)
		utils.RespondWithError(c, app_error.New(err, app_error.ErrCodeResumeInvalidRequest))
		return
	}

	req := entities.DownloadResumeByResumeIdReq{
		ResumeID: resumeId,
	}

	ctx, err := h.authContext.ExtractAuthContext(c)
	if err != nil {
		h.log.ErrorWithID(ctx, "[Handler: DownloadResumeByResumeId] Error getting auth context", err)
		utils.RespondWithError(c, err)
		return
	}

	resp, err := h.resumeService.DownloadResumeByResumeId(ctx, &req)
	if err != nil {
		h.log.ErrorWithID(ctx, "[Handler: DownloadResumeByResumeId] Error downloading resume", err)
		utils.RespondWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}
