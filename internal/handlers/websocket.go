package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
	"gitlab.com/interview-simulation/interview-backend-server/internal/entities"
	app_error "gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
	"gitlab.com/interview-simulation/interview-backend-server/internal/middleware"
	"gitlab.com/interview-simulation/interview-backend-server/internal/services"
	"gitlab.com/interview-simulation/interview-backend-server/internal/utils"
)

type WebSocketHandler struct {
	authContext      middleware.AuthContext
	log              *log.Logger
	webSocketService services.WebSocketService
}

func NewWebSocketHandler(
	authContext middleware.AuthContext,
	log *log.Logger,
	webSocketService services.WebSocketService,
) *WebSocketHandler {
	return &WebSocketHandler{
		authContext:      authContext,
		log:              log,
		webSocketService: webSocketService,
	}
}

func (h *WebSocketHandler) OpenWsConnection(c *gin.Context) {
	ctx := c.Request.Context()
	h.log.InfoWithID(ctx, "[Handler: OpenWsConnection] Called")

	sessionToken := c.Param("id")
	if sessionToken == "" {
		h.log.ErrorWithID(ctx, "[Handler: OpenWsConnection] Session token is required")
		utils.RespondWithError(c, app_error.New(constants.ErrInvalidToken, app_error.ErrCodeSessionInvalidToken))
		return
	}

	var req entities.OpenWsConnectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.log.ErrorWithID(ctx, "[Handler: OpenWsConnection] Error binding request", err)
		utils.RespondWithError(c, err)
		return
	}

	ctx, err := h.authContext.ExtractAuthContext(c)
	if err != nil {
		h.log.ErrorWithID(ctx, "[Handler: OpenWsConnection] Error getting auth context", err)
		utils.RespondWithError(c, err)
		return
	}

	if err = h.webSocketService.OpenWsConnection(ctx, c, &req); err != nil {
		h.log.ErrorWithID(ctx, "[Handler: OpenWsConnection] Error handling web socket", err)
		utils.RespondWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, nil)
}
