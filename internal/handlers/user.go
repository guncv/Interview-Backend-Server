package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gitlab.com/interview-simulation/interview-backend-server/internal/config"
	"gitlab.com/interview-simulation/interview-backend-server/internal/entities"
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

func (h *UserHandler) SignUpUser(c *gin.Context) {
	ctx := c.Request.Context()
	h.log.InfoWithID(ctx, "[Handler: SignUpUser] Called")

	req := &entities.SignUpUserRequest{}
	if err := h.validator.ValidateAndBind(c, req, "SignUpUser"); err != nil {
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

func (h *UserHandler) SendVerifyEmail(c *gin.Context) {
	ctx := c.Request.Context()
	h.log.InfoWithID(ctx, "[Handler: SendVerifyEmail] Called")

	req := &entities.VerifyEmailRequest{}
	if err := h.validator.ValidateAndBind(c, req, "SendVerifyEmail"); err != nil {
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

func (h *UserHandler) ResetVerifyEmailCode(c *gin.Context) {
	ctx := c.Request.Context()
	h.log.InfoWithID(ctx, "[Handler: ResetVerifyEmailCode] Called")

	req := &entities.ResetVerifyEmailCodeRequest{}
	if err := h.validator.ValidateAndBind(c, req, "ResetVerifyEmailCode"); err != nil {
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

func (h *UserHandler) SignInUserByEmailAndPassword(c *gin.Context) {
	ctx := c.Request.Context()
	h.log.InfoWithID(ctx, "[Handler: SignInUserByEmailAndPassword] Called")

	req := &entities.SignInUserByEmailAndPasswordRequest{}
	if err := h.validator.ValidateAndBind(c, req, "SignInUserByEmailAndPassword"); err != nil {
		utils.RespondWithError(c, err)
		return
	}

	res, err := h.userService.SignInUserByEmailAndPassword(ctx, req)
	if err != nil {
		h.log.ErrorWithID(ctx, "[Handler: SignInUserByEmailAndPassword] Error logging in user", err)
		utils.RespondWithError(c, err)
		return
	}

	if err = h.cookies.SetCookie(c, res); err != nil {
		h.log.ErrorWithID(ctx, "[Handler: SignInUserByEmailAndPassword] Error setting cookie", err)
		utils.RespondWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, res)
}

func (h *UserHandler) ForgotPassword(c *gin.Context) {
	ctx := c.Request.Context()
	h.log.InfoWithID(ctx, "[Handler: ForgotPassword] Called")

	req := &entities.ForgotPasswordRequest{}
	if err := h.validator.ValidateAndBind(c, req, "ForgotPassword"); err != nil {
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
