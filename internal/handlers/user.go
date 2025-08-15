package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gitlab.com/interview-simulation/interview-backend-server/internal/config"
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
