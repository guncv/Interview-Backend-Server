package services

import (
	"context"

	"github.com/gin-gonic/gin"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
	ws "gitlab.com/interview-simulation/interview-backend-server/internal/infras/websocket"
	"gitlab.com/interview-simulation/interview-backend-server/internal/middleware"
	"gitlab.com/interview-simulation/interview-backend-server/internal/utils"
)

type WebSocketService interface {
	HandleWebSocket(ctx context.Context, c *gin.Context) (*ws.WebSocketServer, error)
}

type webSocketService struct {
	log             *log.Logger
	authContext     middleware.AuthContext
	generator       utils.Generator
	webSocketServer *ws.WebSocketServer
}

func NewWebSocketService(
	authContext middleware.AuthContext,
	log *log.Logger,
	generator utils.Generator,
	webSocketServer *ws.WebSocketServer,
) WebSocketService {
	return &webSocketService{
		authContext:     authContext,
		log:             log,
		generator:       generator,
		webSocketServer: webSocketServer,
	}
}

func (s *webSocketService) HandleWebSocket(ctx context.Context, c *gin.Context) (*ws.WebSocketServer, error) {
	s.log.InfoWithID(ctx, "[Service: HandleWebSocket] Called")

	authCtx, err := s.authContext.GetAuthContext(ctx)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: HandleWebSocket] Error getting auth context", err)
		return nil, err
	}

	s.webSocketServer.HandleConnection(c.Writer, c.Request, authCtx.Payload.UserID)

	// welcomeMsg := ws.Message{
	// 	Type:      "connection_established",
	// 	Content:   "WebSocket connection established",
	// 	UserID:    authCtx.Payload.UserID,
	// 	Timestamp: time.Now(),
	// }

	// s.webSocketServer.SendOneToOneMessage(ctx, welcomeMsg)

	return s.webSocketServer, nil
}
