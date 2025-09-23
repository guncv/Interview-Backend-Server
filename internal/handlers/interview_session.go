package handlers

import (
	"context"
	"errors"
	"net/http"
	"strconv"

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
// @Failure 400 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Validation error or business logic error"
// @Failure 401 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Unauthorized"
// @Failure 500 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Internal server error"
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
// @Failure 400 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Validation error or business logic error"
// @Failure 401 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Unauthorized"
// @Failure 500 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Internal server error"
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
// @Failure 400 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Invalid session token or request"
// @Failure 401 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Unauthorized"
// @Failure 500 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Internal server error"
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

	session, err := h.interviewSessionService.IsSessionValid(enrichedCtx, &entities.IsSessionValidReq{
		SessionToken: sessionToken,
		UserID:       payload.UserID,
	})
	if err != nil {
		h.log.ErrorWithID(ctx, "[Handler: OpenWsConnection] Error checking session valid", err)
		utils.RespondWithError(c, err)
		return
	}

	if err := h.wsServer.HandleConnection(enrichedCtx, c.Writer, c.Request, session); err != nil {
		h.log.ErrorWithID(ctx, "[Handler: OpenWsConnection] Error handling connection", err)
		utils.RespondWithError(c, app_error.New(err, app_error.ErrCodeAuthInvalidRequest))
		return
	}

	c.Abort()
}

// OpenWsConnection godoc
// @Summary Get chat history by session token
// @Description Get chat history by session token
// @Tags Interview Sessions
// @Accept json
// @Produce json
// @Param session_token path string true "Session token"
// @Security BearerAuth
// @Success 200 {object} entities.GetChatHistoryBySessionTokenResp "Chat history"
// @Failure 400 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Invalid session token or request"
// @Failure 401 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Unauthorized"
// @Failure 500 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Internal server error"
// @Router /sessions/chat-history/{session_token} [get]
func (h *InterviewSessionHandler) GetChatHistoryBySessionToken(c *gin.Context) {
	ctx := c.Request.Context()
	h.log.InfoWithID(ctx, "[Handler: GetChatHistoryBySessionToken] Called")

	sessionToken := c.Param("session_token")
	if sessionToken == "" {
		h.log.ErrorWithID(ctx, "[Handler: GetChatHistoryBySessionToken] Session token is required")
		utils.RespondWithError(c, app_error.New(errors.New("session token is required"), app_error.ErrCodeSessionInvalidToken))
		return
	}

	if err := h.validator.GetValidate().Var(sessionToken, "required,uuid"); err != nil {
		h.log.ErrorWithID(ctx, "[Handler: GetChatHistoryBySessionToken] Invalid session token", err)
		utils.RespondWithError(c, app_error.New(err, app_error.ErrCodeSessionInvalidToken))
		return
	}

	ctx, err := h.authContext.ExtractAuthContext(c)
	if err != nil {
		h.log.ErrorWithID(ctx, "[Handler: GetChatHistoryBySessionToken] Error getting auth context", err)
		utils.RespondWithError(c, err)
		return
	}

	req := entities.GetChatHistoryBySessionTokenReq{
		SessionToken: sessionToken,
	}

	resp, err := h.interviewSessionService.GetChatHistoryBySessionToken(ctx, &req)
	if err != nil {
		h.log.ErrorWithID(ctx, "[Handler: GetChatHistoryBySessionToken] Error getting chat history", err)
		utils.RespondWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// ListInterviewSessionsByUserIDWithCursor godoc
// @Summary List interview sessions by user ID with cursor pagination
// @Description Get a paginated list of interview sessions for the authenticated user with optional search and filtering
// @Tags Interview Sessions
// @Accept json
// @Produce json
// @Param search_text query string false "Search text to filter sessions by resume name or position"
// @Param status query string false "Filter sessions by status"
// @Param cursor_id query string false "Cursor ID for pagination"
// @Param cursor_created_at query string false "Cursor created_at for pagination (RFC3339 format)"
// @Param limit query int false "Number of sessions to return (default: 20)"
// @Param type query string false "Type of pagination (next or prev)"
// @Security BearerAuth
// @Success 200 {object} entities.ListInterviewSessionsByUserIDResp "List of interview sessions"
// @Failure 400 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Validation error or business logic error"
// @Failure 401 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Unauthorized"
// @Failure 500 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Internal server error"
// @Router /sessions/cursor [get]
func (h *InterviewSessionHandler) ListInterviewSessionsByUserIDWithCursor(c *gin.Context) {
	ctx := c.Request.Context()
	h.log.InfoWithID(ctx, "[Handler: ListInterviewSessionsByUserID] Called")

	req := entities.ListInterviewSessionsByUserIDWithCursorReq{}

	if searchText := c.Query("search_text"); searchText != "" {
		req.SearchText = &searchText
	} else {
		req.SearchText = nil
	}

	if status := c.Query("status"); status != "" {
		req.Status = &status
	} else {
		req.Status = nil
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil {
			h.log.ErrorWithID(ctx, "[Handler: ListInterviewSessionsByUserID] Invalid limit", err)
			utils.RespondWithError(c, app_error.New(err, app_error.ErrCodeGeneralInvalidLimit))
			return
		}
		if limit > 0 {
			req.Limit = &limit
		} else {
			req.Limit = nil
		}
	} else {
		req.Limit = nil
	}

	cursorID := c.Query("cursor_id")
	cursorCreatedAt := c.Query("cursor_created_at")
	if cursorID != "" && cursorCreatedAt != "" {
		req.Cursor = &entities.Cursor{
			ID:        cursorID,
			CreatedAt: cursorCreatedAt,
		}
	} else {
		req.Cursor = nil
	}

	if typeParam := c.Query("type"); typeParam != "" {
		req.Type = &typeParam
	} else {
		req.Type = nil
	}

	ctx, err := h.authContext.ExtractAuthContext(c)
	if err != nil {
		h.log.ErrorWithID(ctx, "[Handler: ListInterviewSessionsByUserID] Error getting auth context", err)
		utils.RespondWithError(c, err)
		return
	}

	resp, err := h.interviewSessionService.ListInterviewSessionsByUserIDWithCursor(ctx, &req)
	if err != nil {
		h.log.ErrorWithID(ctx, "[Handler: ListInterviewSessionsByUserID] Error listing interview sessions", err)
		utils.RespondWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// ListInterviewSessionsByUserIDWithJumpPagination godoc
// @Summary List interview sessions by user ID with jump pagination
// @Description Get a paginated list of interview sessions for the authenticated user with optional search and filtering
// @Tags Interview Sessions
// @Accept json
// @Produce json
// @Param search_text query string false "Search text to filter sessions by resume name or position"
// @Param status query string false "Filter sessions by status"
// @Param offset query int false "Offset for pagination"
// @Param limit query int false "Number of sessions to return (default: 20)"
// @Security BearerAuth
// @Success 200 {object} entities.ListInterviewSessionsByUserIDResp "List of interview sessions"
// @Failure 400 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Validation error or business logic error"
// @Failure 401 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Unauthorized"
// @Failure 500 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Internal server error"
// @Router /sessions/jump [get]
func (h *InterviewSessionHandler) ListInterviewSessionsByUserIDWithJumpPagination(c *gin.Context) {
	ctx := c.Request.Context()
	h.log.InfoWithID(ctx, "[Handler: ListInterviewSessionsByUserID] Called")

	req := entities.ListInterviewSessionsByUserIDWithJumpPaginationReq{}

	if searchText := c.Query("search_text"); searchText != "" {
		req.SearchText = &searchText
	} else {
		req.SearchText = nil
	}

	if status := c.Query("status"); status != "" {
		req.Status = &status
	} else {
		req.Status = nil
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil {
			h.log.ErrorWithID(ctx, "[Handler: ListInterviewSessionsByUserID] Invalid limit", err)
			utils.RespondWithError(c, app_error.New(err, app_error.ErrCodeGeneralInvalidLimit))
			return
		}
		if limit > 0 {
			req.Limit = &limit
		} else {
			req.Limit = nil
		}
	} else {
		req.Limit = nil
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		offset, err := strconv.Atoi(offsetStr)
		if err != nil {
			h.log.ErrorWithID(ctx, "[Handler: ListInterviewSessionsByUserID] Invalid offset", err)
			utils.RespondWithError(c, app_error.New(err, app_error.ErrCodeGeneralInvalidOffset))
			return
		}
		if offset > 0 {
			req.Offset = &offset
		} else {
			req.Offset = nil
		}
	} else {
		req.Offset = nil
	}

	ctx, err := h.authContext.ExtractAuthContext(c)
	if err != nil {
		h.log.ErrorWithID(ctx, "[Handler: ListInterviewSessionsByUserID] Error getting auth context", err)
		utils.RespondWithError(c, err)
		return
	}

	resp, err := h.interviewSessionService.ListInterviewSessionsByUserIDWithJumpPagination(ctx, &req)
	if err != nil {
		h.log.ErrorWithID(ctx, "[Handler: ListInterviewSessionsByUserID] Error listing interview sessions", err)
		utils.RespondWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// DeleteUserInterviewSessionByID godoc
// @Summary Delete interview session by ID
// @Description Delete an interview session by ID
// @Tags Interview Sessions
// @Accept json
// @Produce json
// @Param session_id path string true "Session ID"
// @Security BearerAuth
// @Success 204 "Interview session deleted successfully"
// @Failure 400 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Validation error or business logic error"
// @Failure 401 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Unauthorized"
// @Failure 500 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Internal server error"
// @Router /sessions/{session_id} [delete]
func (h *InterviewSessionHandler) DeleteUserInterviewSessionByID(c *gin.Context) {
	ctx := c.Request.Context()
	h.log.InfoWithID(ctx, "[Handler: DeleteUserInterviewSessionByID] Called")

	sessionID := c.Param("session_id")
	if sessionID == "" {
		h.log.ErrorWithID(ctx, "[Handler: DeleteUserInterviewSessionByID] Session ID is required")
		utils.RespondWithError(c, app_error.New(errors.New("session ID is required"), app_error.ErrCodeSessionInvalidSessionID))
		return
	}

	if err := h.validator.GetValidate().Var(sessionID, "required,uuid"); err != nil {
		h.log.ErrorWithID(ctx, "[Handler: DeleteUserInterviewSessionByID] Invalid session ID", err)
		utils.RespondWithError(c, app_error.New(err, app_error.ErrCodeSessionInvalidSessionID))
		return
	}

	ctx, err := h.authContext.ExtractAuthContext(c)
	if err != nil {
		h.log.ErrorWithID(ctx, "[Handler: DeleteUserInterviewSessionByID] Error getting auth context", err)
		utils.RespondWithError(c, err)
		return
	}

	if err := h.interviewSessionService.DeleteUserInterviewSessionByID(ctx, sessionID); err != nil {
		h.log.ErrorWithID(ctx, "[Handler: DeleteUserInterviewSessionByID] Error deleting interview session", err)
		utils.RespondWithError(c, err)
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// GetInterviewSessionInformationByID godoc
// @Summary Get interview session information by ID
// @Description Get interview session information by ID
// @Tags Interview Sessions
// @Accept json
// @Produce json
// @Param session_id path string true "Session ID"
// @Security BearerAuth
// @Success 200 {object} entities.GetInterviewSessionInformationResp "Interview session information"
// @Failure 400 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Validation error or business logic error"
// @Failure 401 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Unauthorized"
// @Failure 500 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Internal server error"
// @Router /sessions/{session_id} [get]
func (h *InterviewSessionHandler) GetInterviewSessionInformationByID(c *gin.Context) {
	ctx := c.Request.Context()
	h.log.InfoWithID(ctx, "[Handler: GetInterviewSessionInformationByID] Called")

	sessionID := c.Param("session_id")
	if sessionID == "" {
		h.log.ErrorWithID(ctx, "[Handler: GetInterviewSessionInformationByID] Session ID is required")
		utils.RespondWithError(c, app_error.New(errors.New("session ID is required"), app_error.ErrCodeSessionInvalidSessionID))
		return
	}

	if err := h.validator.GetValidate().Var(sessionID, "required,uuid"); err != nil {
		h.log.ErrorWithID(ctx, "[Handler: GetInterviewSessionInformationByID] Invalid session ID", err)
		utils.RespondWithError(c, app_error.New(err, app_error.ErrCodeSessionInvalidSessionID))
		return
	}

	ctx, err := h.authContext.ExtractAuthContext(c)
	if err != nil {
		h.log.ErrorWithID(ctx, "[Handler: GetInterviewSessionInformationByID] Error getting auth context", err)
		utils.RespondWithError(c, err)
		return
	}

	resp, err := h.interviewSessionService.GetInterviewSessionInformationByID(ctx, sessionID)
	if err != nil {
		h.log.ErrorWithID(ctx, "[Handler: GetInterviewSessionInformationByID] Error getting interview session information", err)
		utils.RespondWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}
