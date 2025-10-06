package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"gitlab.com/interview-simulation/interview-backend-server/internal/config"
	"gitlab.com/interview-simulation/interview-backend-server/internal/entities"
	app_error "gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
	"gitlab.com/interview-simulation/interview-backend-server/internal/middleware"
	"gitlab.com/interview-simulation/interview-backend-server/internal/services"
	"gitlab.com/interview-simulation/interview-backend-server/internal/utils"
)

type UserHandler struct {
	userService services.UserService
	log         *log.Logger
	config      *config.Config
	authContext middleware.AuthContext
	validator   utils.Validator
	cookies     utils.Cookies
}

func NewUserHandler(
	l *log.Logger,
	userService services.UserService,
	config *config.Config,
	authContext middleware.AuthContext,
	validator utils.Validator,
	cookies utils.Cookies) *UserHandler {
	return &UserHandler{
		log:         l,
		userService: userService,
		config:      config,
		authContext: authContext,
		validator:   validator,
		cookies:     cookies,
	}
}

// HealthCheck godoc
// @Summary Health check endpoint
// @Description Check if the server is running and healthy
// @Tags Health Check
// @Accept json
// @Produce json
// @Success 200 {object} entities.HealthCheckResponse
// @Router /auth/health [get]
func (h *UserHandler) HealthCheck(c *gin.Context) {
	ctx := c.Request.Context()

	res, err := h.userService.HealthCheck(ctx)
	if err != nil {
		h.log.ErrorWithID(ctx, "[Handler: HealthCheck] Error checking health", err)
		utils.RespondWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, res)
}

// GetGoogleAuthURL godoc
// @Summary Get Google OAuth URL
// @Description Generate and return Google OAuth authorization URL
// @Tags Auth
// @Accept json
// @Produce json
// @Success 200 {object} entities.GoogleAuthURLResponse "OAuth URL response"
// @Failure 500 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Internal server error"
// @Router /auth/google/url [get]
func (h *UserHandler) GetGoogleAuthURL(c *gin.Context) {
	ctx := c.Request.Context()

	resp, err := h.userService.GetGoogleAuthURL(ctx)
	if err != nil {
		h.log.ErrorWithID(ctx, "[Handler: GetGoogleAuthURL] Error generating auth URL", err)
		utils.RespondWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// HandleGoogleCallback godoc
// @Summary Handle Google OAuth callback
// @Description Process Google OAuth callback and return authentication tokens
// @Tags Auth
// @Accept json
// @Produce json
// @Param code query string true "Authorization code from Google"
// @Param state query string true "State from Google"
// @Success 200 {object} entities.HandleGoogleCallbackResp "Authentication response"
// @Failure 400 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Bad request"
// @Failure 500 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Internal server error"
// @Router /auth/google/callback [get]
func (h *UserHandler) HandleGoogleCallback(c *gin.Context) {
	ctx := c.Request.Context()

	code := c.Query("code")
	state := c.Query("state")
	if code == "" || state == "" {
		h.log.ErrorWithID(ctx, "[Handler: HandleGoogleCallback] Error getting code or state", errors.New("missing code or state"))
		utils.RespondWithError(c, app_error.New(errors.New("missing code or state"), app_error.ErrCodeAuthInvalidRequest))
		return
	}

	req := &entities.HandleGoogleCallbackReq{
		Code:  code,
		State: state,
	}

	resp, err := h.userService.HandleGoogleCallback(ctx, req)
	if err != nil {
		h.log.ErrorWithID(ctx, "[Handler: HandleGoogleCallback] Error handling Google callback", err)
		utils.RespondWithError(c, err)
		return
	}

	h.cookies.SetRefreshTokenCookie(c, resp.RefreshToken)
	c.JSON(http.StatusOK, resp)
}

// GetFacebookAuthURL godoc
// @Summary Get Facebook OAuth URL
// @Description Generate and return Facebook OAuth authorization URL
// @Tags Auth
// @Accept json
// @Produce json
// @Success 200 {object} entities.FacebookAuthURLResponse "OAuth URL response"
// @Failure 500 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Internal server error"
// @Router /auth/facebook/url [get]
func (h *UserHandler) GetFacebookAuthURL(c *gin.Context) {
	ctx := c.Request.Context()

	resp, err := h.userService.GetFacebookAuthURL(ctx)
	if err != nil {
		h.log.ErrorWithID(ctx, "[Handler: GetFacebookAuthURL] Error generating auth URL", err)
		utils.RespondWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// HandleFacebookCallback godoc
// @Summary Handle Facebook OAuth callback
// @Description Process Facebook OAuth callback and return authentication tokens
// @Tags Auth
// @Accept json
// @Produce json
// @Param code query string true "Authorization code from Facebook"
// @Param state query string true "State from Facebook"
// @Success 200 {object} entities.HandleFacebookCallbackResp "Authentication response"
// @Failure 400 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Bad request"
// @Failure 500 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Internal server error"
// @Router /auth/facebook/callback [get]
func (h *UserHandler) HandleFacebookCallback(c *gin.Context) {
	ctx := c.Request.Context()

	code := c.Query("code")
	state := c.Query("state")
	if code == "" || state == "" {
		h.log.ErrorWithID(ctx, "[Handler: HandleFacebookCallback] Error getting code or state", errors.New("missing code or state"))
		utils.RespondWithError(c, app_error.New(errors.New("missing code or state"), app_error.ErrCodeAuthInvalidRequest))
		return
	}

	req := &entities.HandleFacebookCallbackReq{
		Code:  code,
		State: state,
	}

	resp, err := h.userService.HandleFacebookCallback(ctx, req)
	if err != nil {
		h.log.ErrorWithID(ctx, "[Handler: HandleFacebookCallback] Error handling Facebook callback", err)
		utils.RespondWithError(c, err)
		return
	}

	h.cookies.SetRefreshTokenCookie(c, resp.RefreshToken)
	c.JSON(http.StatusOK, resp)
}

// SignOut godoc
// @Summary User logout
// @Description Sign out user and invalidate tokens
// @Tags Auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 204 "User signed out successfully"
// @Failure 401 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Unauthorized"
// @Failure 500 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Internal server error"
// @Router /auth/sign-out [post]
func (h *UserHandler) SignOut(c *gin.Context) {
	ctx, err := h.authContext.ExtractAuthContext(c)
	if err != nil {
		h.log.ErrorWithID(ctx, "[Handler: SignOut] Error getting auth context", err)
		utils.RespondWithError(c, err)
		return
	}

	err = h.userService.SignOut(ctx)
	if err != nil {
		h.log.ErrorWithID(ctx, "[Handler: SignOut] Error signing out", err)
		utils.RespondWithError(c, err)
		return
	}

	h.cookies.ClearRefreshTokenCookie(c)
	c.JSON(http.StatusNoContent, nil)
}
