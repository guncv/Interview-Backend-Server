package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
	"gitlab.com/interview-simulation/interview-backend-server/internal/entities"
	app_error "gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/websocket"
	"gitlab.com/interview-simulation/interview-backend-server/internal/middleware"
	"gitlab.com/interview-simulation/interview-backend-server/internal/services"
	"gitlab.com/interview-simulation/interview-backend-server/internal/utils"
)

type InterviewSessionHandler struct {
	interviewSessionService services.InterviewSessionService
	log                     *log.Logger
	authContext             middleware.AuthContext
	validator               utils.Validator
	wsServer                websocket.WebSocketServerInterface
	jwtToken                utils.JwtToken
	middle                  middleware.AuthMiddleware
}

func NewInterviewSessionHandler(
	interviewSessionService services.InterviewSessionService,
	log *log.Logger,
	authContext middleware.AuthContext,
	validator utils.Validator,
	wsServer websocket.WebSocketServerInterface,
	jwtToken utils.JwtToken,
	middle middleware.AuthMiddleware,
) *InterviewSessionHandler {
	return &InterviewSessionHandler{
		interviewSessionService: interviewSessionService,
		log:                     log,
		authContext:             authContext,
		validator:               validator,
		wsServer:                wsServer,
		jwtToken:                jwtToken,
		middle:                  middle,
	}
}

// CreateInterviewSessionWithNewResume godoc
// @Summary Create interview session with new resume
// @Description Create a new interview session with a newly uploaded resume
// @Tags Interview Sessions
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Resume file (PDF, DOC, DOCX)"
// @Param position formData string true "Job position"
// @Param company formData string true "Company name"
// @Param work_type formData string true "Work type (full-time, part-time, contract, etc.)"
// @Param job_requirements formData string true "Job requirements description"
// @Param interview_type formData string true "Interview type (technical, behavioral, etc.)"
// @Param language formData string true "Interview language"
// @Param is_consent formData boolean true "User consent for interview"
// @Security BearerAuth
// @Success 201 {object} entities.CreateInterviewSessionWithNewResumeResponse
// @Failure 400 {object} app_error.AppError "Validation error or business logic error"
// @Failure 401 {object} app_error.AppError "Unauthorized"
// @Failure 500 {object} app_error.AppError "Internal server error"
// @Router /sessions [post]
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

// CreateInterviewSessionWithExistingResume godoc
// @Summary Create interview session with existing resume
// @Description Create a new interview session using an existing resume
// @Tags Interview Sessions
// @Accept json
// @Produce json
// @Param request body entities.CreateInterviewSessionWithExistingResumeReq true "Interview session creation request with existing resume"
// @Security BearerAuth
// @Success 201 {object} entities.CreateInterviewSessionWithExistingResumeResp
// @Failure 400 {object} app_error.AppError "Validation error or business logic error"
// @Failure 401 {object} app_error.AppError "Unauthorized"
// @Failure 500 {object} app_error.AppError "Internal server error"
// @Router /sessions/existing [post]
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

// OpenWsConnection godoc
// @Summary Open WebSocket connection
// @Description Establish a WebSocket connection for real-time interview communication
// @Tags Interview Sessions
// @Accept json
// @Produce json
// @Param id path string true "Session token"
// @Param access_token query string true "Access token for authentication"
// @Success 200 "WebSocket connection established"
// @Failure 400 {object} app_error.AppError "Invalid session token or request"
// @Failure 401 {object} app_error.AppError "Unauthorized"
// @Failure 500 {object} app_error.AppError "Internal server error"
// @Router /ws/connect/{id} [get]
func (h *InterviewSessionHandler) OpenWsConnection(c *gin.Context) {
	ctx := c.Request.Context()
	h.log.InfoWithID(ctx, "[Handler: OpenWsConnection] Called")

	sessionToken := c.Param("id")
	if err := h.validator.GetValidate().Var(sessionToken, "required,uuid"); err != nil {
		h.log.ErrorWithID(ctx, "[Handler: OpenWsConnection] Invalid session token", err)
		utils.RespondWithError(c, app_error.New(err, app_error.ErrCodeSessionInvalidToken))
		return
	}

	accessToken := c.Query("access_token")
	if err := h.validator.GetValidate().Var(accessToken, "required"); err != nil {
		h.log.ErrorWithID(ctx, "[Handler: OpenWsConnection] Invalid access token", err)
		utils.RespondWithError(c, app_error.New(err, app_error.ErrCodeAuthInvalidHeader))
		return
	}

	payload, err := h.middle.VerifyAndRenewAccessToken(c, accessToken)
	if err != nil {
		h.log.ErrorWithID(ctx, "[Handler: OpenWsConnection] Authentication failed", err)
		utils.RespondWithError(c, err)
		return
	}

	enrichedCtx := context.WithValue(ctx, constants.AuthContextKey, &middleware.AuthPayload{
		Payload: payload,
	})

	req := entities.OpenWsConnectionRequest{
		SessionToken: sessionToken,
		UserID:       payload.UserID,
	}

	if err := h.wsServer.HandleConnection(enrichedCtx, c.Writer, c.Request, req); err != nil {
		h.log.ErrorWithID(ctx, "[Handler: OpenWsConnection] Error handling connection", err)
		utils.RespondWithError(c, app_error.New(err, app_error.ErrCodeAuthInvalidRequest))
		return
	}

	c.Abort()
}
