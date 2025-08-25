package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gitlab.com/interview-simulation/interview-backend-server/internal/config"
	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
	"gitlab.com/interview-simulation/interview-backend-server/internal/entities"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
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
	h.log.InfoWithID(ctx, "[Handler: HealthCheck] Called")

	res, err := h.userService.HealthCheck(ctx)
	if err != nil {
		h.log.ErrorWithID(ctx, "[Handler: HealthCheck] Error checking health", err)
		utils.RespondWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, res)
}

// SignUpUser godoc
// @Summary User registration
// @Description Create a new user account with email verification
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body entities.SignUpUserRequest true "User registration details"
// @Success 200 {object} entities.SignUpUserResponse
// @Failure 400 {object} app_error.AppError "Validation error or business logic error"
// @Failure 500 {object} app_error.AppError "Internal server error"
// @Router /auth/sign-up [post]
func (h *UserHandler) SignUpUser(c *gin.Context) {
	ctx := c.Request.Context()
	h.log.InfoWithID(ctx, "[Handler: SignUpUser] Called")

	req := &entities.SignUpUserRequest{}
	if err := h.validator.ValidateAndBind(c, req, "SignUpUser"); err != nil {
		h.log.ErrorWithID(ctx, "[Handler: SignUpUser] Error validate and bind", err)
		utils.RespondWithError(c, err)
		return
	}

	resp, err := h.userService.SignUpUser(ctx, req)
	if err != nil {
		h.log.ErrorWithID(ctx, "[Handler: SignUpUser] Error sign up user", err)
		utils.RespondWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// SendVerifyEmail godoc
// @Summary Send email verification
// @Description Send verification code to user's email address
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body entities.VerifyEmailRequest true "Email verification details"
// @Success 204 "Email sent successfully"
// @Failure 400 {object} app_error.AppError "Validation error or business logic error"
// @Failure 500 {object} app_error.AppError "Internal server error"
// @Router /auth/verify-email [post]
func (h *UserHandler) SendVerifyEmail(c *gin.Context) {
	ctx := c.Request.Context()
	h.log.InfoWithID(ctx, "[Handler: SendVerifyEmail] Called")

	req := &entities.VerifyEmailRequest{}
	if err := h.validator.ValidateAndBind(c, req, "SendVerifyEmail"); err != nil {
		h.log.ErrorWithID(ctx, "[Handler: SendVerifyEmail] Error validate and bind", err)
		utils.RespondWithError(c, err)
		return
	}

	err := h.userService.SendVerifyEmail(ctx, req)
	if err != nil {
		h.log.ErrorWithID(ctx, "[Handler: SendVerifyEmail] Error send verify email", err)
		utils.RespondWithError(c, err)
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// ResetVerifyEmailCode godoc
// @Summary Reset email verification code
// @Description Generate a new verification code for email verification
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body entities.ResetVerifyEmailCodeRequest true "Reset verification code request"
// @Success 200 {object} entities.ResetVerifyEmailCodeResponse
// @Failure 400 {object} app_error.AppError "Validation error or business logic error"
// @Failure 500 {object} app_error.AppError "Internal server error"
// @Router /auth/reset-verify-email [post]
func (h *UserHandler) ResetVerifyEmailCode(c *gin.Context) {
	ctx := c.Request.Context()
	h.log.InfoWithID(ctx, "[Handler: ResetVerifyEmailCode] Called")

	req := &entities.ResetVerifyEmailCodeRequest{}
	if err := h.validator.ValidateAndBind(c, req, "ResetVerifyEmailCode"); err != nil {
		h.log.ErrorWithID(ctx, "[Handler: ResetVerifyEmailCode] Error validate and bind", err)
		utils.RespondWithError(c, err)
		return
	}

	resp, err := h.userService.ResetVerifyEmailCode(ctx, req)
	if err != nil {
		h.log.ErrorWithID(ctx, "[Handler: ResetVerifyEmailCode] Error reset verify email code", err)
		utils.RespondWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// SignInUserByEmailAndPassword godoc
// @Summary User authentication
// @Description Authenticate user with email and password, returns access and refresh tokens
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body entities.SignInUserByEmailAndPasswordRequest true "Login credentials"
// @Success 200 {object} entities.SignInUserByEmailAndPasswordResponse
// @Failure 400 {object} app_error.AppError "Validation error or invalid credentials"
// @Failure 500 {object} app_error.AppError "Internal server error"
// @Router /auth/sign-in [post]
func (h *UserHandler) SignInUserByEmailAndPassword(c *gin.Context) {
	ctx := c.Request.Context()
	h.log.InfoWithID(ctx, "[Handler: SignInUserByEmailAndPassword] Called")

	req := &entities.SignInUserByEmailAndPasswordRequest{}
	if err := h.validator.ValidateAndBind(c, req, "SignInUserByEmailAndPassword"); err != nil {
		h.log.ErrorWithID(ctx, "[Handler: SignInUserByEmailAndPassword] Error validate and bind", err)
		utils.RespondWithError(c, err)
		return
	}

	res, err := h.userService.SignInUserByEmailAndPassword(ctx, req)
	if err != nil {
		h.log.ErrorWithID(ctx, "[Handler: SignInUserByEmailAndPassword] Error logging in user", err)
		utils.RespondWithError(c, err)
		return
	}

	h.cookies.SetRefreshTokenCookie(c, res.RefreshToken)

	c.JSON(http.StatusOK, res)
}

// ForgotPassword godoc
// @Summary Request password reset
// @Description Send password reset link to user's email
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body entities.ForgotPasswordRequest true "Password reset request"
// @Success 204 "Password reset email sent successfully"
// @Failure 400 {object} app_error.AppError "Validation error or business logic error"
// @Failure 500 {object} app_error.AppError "Internal server error"
// @Router /auth/forgot-password [post]
func (h *UserHandler) ForgotPassword(c *gin.Context) {
	ctx := c.Request.Context()
	h.log.InfoWithID(ctx, "[Handler: ForgotPassword] Called")

	req := &entities.ForgotPasswordRequest{}
	if err := h.validator.ValidateAndBind(c, req, "ForgotPassword"); err != nil {
		h.log.ErrorWithID(ctx, "[Handler: ForgotPassword] Error validate and bind", err)
		utils.RespondWithError(c, err)
		return
	}

	err := h.userService.ForgotPassword(ctx, req)
	if err != nil {
		h.log.ErrorWithID(ctx, "[Handler: ForgotPassword] Error forgot password", err)
		utils.RespondWithError(c, err)
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// ResetUserPassword godoc
// @Summary Reset user password
// @Description Reset user password using reset token
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body entities.ResetUserPasswordRequest true "Password reset details"
// @Success 204 "Password reset successfully"
// @Failure 400 {object} app_error.AppError "Validation error or invalid token"
// @Failure 500 {object} app_error.AppError "Internal server error"
// @Router /auth/reset-password [post]
func (h *UserHandler) ResetUserPassword(c *gin.Context) {
	ctx := c.Request.Context()
	h.log.InfoWithID(ctx, "[Handler: ResetUserPassword] Called")

	req := &entities.ResetUserPasswordRequest{}
	if err := h.validator.ValidateAndBind(c, req, "ResetUserPassword"); err != nil {
		h.log.ErrorWithID(ctx, "[Handler: ResetUserPassword] Error validate and bind", err)
		utils.RespondWithError(c, err)
		return
	}

	err := h.userService.ResetUserPassword(ctx, req)
	if err != nil {
		h.log.ErrorWithID(ctx, "[Handler: ResetUserPassword] Error resetting user password", err)
		utils.RespondWithError(c, err)
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// RefreshToken godoc
// @Summary Refresh access token
// @Description Get new access token using refresh token from cookie
// @Tags Auth
// @Accept json
// @Produce json
// @Success 200 {object} entities.RefreshTokenResponse
// @Failure 400 {object} app_error.AppError "Invalid refresh token"
// @Failure 500 {object} app_error.AppError "Internal server error"
// @Router /auth/refresh-token [post]
func (h *UserHandler) RefreshToken(c *gin.Context) {
	ctx := c.Request.Context()
	h.log.InfoWithID(ctx, "[Handler: RefreshToken] Called")

	cookie, err := c.Request.Cookie(string(constants.RefreshTokenCookieKey))
	if err != nil {
		h.log.ErrorWithID(ctx, "[Handler: RefreshToken] Error getting refresh token from cookie", err)
		utils.RespondWithError(c, app_error.New(err, app_error.ErrCodeAuthInvalidRefreshToken))
		return
	}

	res, err := h.userService.RefreshToken(ctx, &entities.RefreshTokenRequest{
		RefreshToken: cookie.Value,
	})
	if err != nil {
		h.log.ErrorWithID(ctx, "[Handler: RefreshToken] Error refreshing token", err)
		utils.RespondWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, res)
}

// SignOut godoc
// @Summary User logout
// @Description Sign out user and invalidate tokens
// @Tags Auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 204 "User signed out successfully"
// @Failure 401 {object} app_error.AppError "Unauthorized"
// @Failure 500 {object} app_error.AppError "Internal server error"
// @Router /auth/sign-out [post]
func (h *UserHandler) SignOut(c *gin.Context) {
	ctx := c.Request.Context()
	h.log.InfoWithID(ctx, "[Handler: SignOut] Called")

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
