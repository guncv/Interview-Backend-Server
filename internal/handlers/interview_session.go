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

type InterviewSessionHandler struct {
	interviewSessionService services.InterviewSessionService
	log                     *log.Logger
	authContext             middleware.AuthContext
	validator               utils.Validator
}

func NewInterviewSessionHandler(
	interviewSessionService services.InterviewSessionService,
	log *log.Logger,
	authContext middleware.AuthContext,
	validator utils.Validator,
) *InterviewSessionHandler {
	return &InterviewSessionHandler{
		interviewSessionService: interviewSessionService,
		log:                     log,
		authContext:             authContext,
		validator:               validator,
	}
}

func (h *InterviewSessionHandler) CreateInterviewSessionWithNewResume(c *gin.Context) {
	ctx := c.Request.Context()
	h.log.InfoWithID(ctx, "[Handler: CreateInterviewSessionWithNewResume] Called")

	var req entities.CreateInterviewSessionWithNewResumeRequest
	if err := h.validator.ValidateAndBind(c, &req, "CreateInterviewSessionWithNewResume"); err != nil {
		utils.RespondWithError(c, err)
		return
	}

	ctx, err := h.authContext.ExtractAuthContext(c)
	if err != nil {
		h.log.ErrorWithID(ctx, "[Handler: CreateInterviewSessionWithNewResume] Error getting auth context", err)
		utils.RespondWithError(c, err)
		return
	}

	resp, err := h.interviewSessionService.CreateInterviewSessionWithNewResume(ctx, &req)
	if err != nil {
		h.log.ErrorWithID(ctx, "[Handler: CreateInterviewSessionWithNewResume] Error creating interview session", err)
		utils.RespondWithError(c, err)
		return
	}

	c.JSON(http.StatusCreated, resp)
}

func (h *InterviewSessionHandler) CreateInterviewSessionWithExistingResume(c *gin.Context) {
	ctx := c.Request.Context()
	h.log.InfoWithID(ctx, "[Handler: CreateInterviewSessionWithExistingResume] Called")

	var req entities.CreateInterviewSessionWithExistingResumeReq
	if err := h.validator.ValidateAndBind(c, &req, "CreateInterviewSessionWithExistingResume"); err != nil {
		utils.RespondWithError(c, err)
		return
	}

	ctx, err := h.authContext.ExtractAuthContext(c)
	if err != nil {
		h.log.ErrorWithID(ctx, "[Handler: CreateInterviewSessionWithExistingResume] Error getting auth context", err)
		utils.RespondWithError(c, err)
		return
	}

	resp, err := h.interviewSessionService.CreateInterviewSessionWithExistingResume(ctx, &req)
	if err != nil {
		h.log.ErrorWithID(ctx, "[Handler: CreateInterviewSessionWithExistingResume] Error creating interview session", err)
		utils.RespondWithError(c, err)
		return
	}

	c.JSON(http.StatusCreated, resp)
}
