package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"gitlab.com/interview-simulation/interview-backend-server/internal/config"
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

func (h *UserHandler) SignInWithGoogle(c *gin.Context) {
	ctx := c.Request.Context()

	code := c.Query("code")
	if code == "" {
		h.log.ErrorWithID(ctx, "[Handler: SignInWithGoogle] Error getting code", errors.New("missing code"))
		utils.RespondWithError(c, app_error.New(errors.New("missing code"), app_error.ErrCodeAuthInvalidRequest))
		return
	}

	// userInfo, err := h.userService.HandleGoogleCallback(c, code)
	// if err != nil {
	// 	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	// 	return
	// }

	// c.JSON(http.StatusOK, gin.H{"message": "Login success", "user": userInfo})
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
