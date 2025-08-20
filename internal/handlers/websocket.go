package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
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

func (h *WebSocketHandler) HandleWebSocket(c *gin.Context) {
	ctx := c.Request.Context()
	h.log.InfoWithID(ctx, "[Handler: HandleWebSocket] Called")

	ctx, err := h.authContext.ExtractAuthContext(c)
	if err != nil {
		h.log.ErrorWithID(ctx, "[Handler: HandleWebSocket] Error getting auth context", err)
		utils.RespondWithError(c, err)
		return
	}

	resp, err := h.webSocketService.HandleWebSocket(ctx, c)
	if err != nil {
		h.log.ErrorWithID(ctx, "[Handler: HandleWebSocket] Error handling web socket", err)
		utils.RespondWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}
