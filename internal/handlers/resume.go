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

func (h *ResumeHandler) CreateResume(c *gin.Context) {
	ctx := c.Request.Context()
	h.log.InfoWithID(ctx, "[Handler: CreateResume] Called")

	var req entities.CreateResumeWithRequirementsRequest
	if err := h.validator.ValidateAndBind(c, &req, "CreateResume"); err != nil {
		utils.RespondWithError(c, err)
		return
	}

	ctx, err := h.authContext.ExtractAuthContext(c)
	if err != nil {
		h.log.ErrorWithID(ctx, "[Handler: CreateResume] Error getting auth context", err)
		utils.RespondWithError(c, err)
		return
	}

	err = h.resumeService.CreateResumeWithRequirements(ctx, &req)
	if err != nil {
		h.log.ErrorWithID(ctx, "[Handler: CreateResume] Error creating resume", err)
		utils.RespondWithError(c, err)
		return
	}

	c.JSON(http.StatusCreated, nil)
}

func (h *ResumeHandler) ListResume(c *gin.Context) {
	ctx := c.Request.Context()
	h.log.InfoWithID(ctx, "[Handler: ListResume] Called")

	var req entities.ListResumeRequest
	if err := h.validator.ValidateAndBind(c, &req, "ListResume"); err != nil {
		utils.RespondWithError(c, err)
		return
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

func (h *ResumeHandler) SwitchDefaultResume(c *gin.Context) {
	ctx := c.Request.Context()
	h.log.InfoWithID(ctx, "[Handler: SwitchDefaultResume] Called")

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
