package services

import (
	"context"
	"encoding/json"

	"github.com/gin-gonic/gin"
	"gitlab.com/interview-simulation/interview-backend-server/internal/entities"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/database"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
	ws "gitlab.com/interview-simulation/interview-backend-server/internal/infras/websocket"
	"gitlab.com/interview-simulation/interview-backend-server/internal/middleware"
	"gitlab.com/interview-simulation/interview-backend-server/internal/utils"
)

type WebSocketService interface {
	OpenWsConnection(ctx context.Context, c *gin.Context, req *entities.OpenWsConnectionRequest) error
}

type webSocketService struct {
	log             *log.Logger
	authContext     middleware.AuthContext
	generator       utils.Generator
	webSocketServer *ws.WebSocketServer
	redisClient     database.RedisClient
}

func NewWebSocketService(
	authContext middleware.AuthContext,
	log *log.Logger,
	generator utils.Generator,
	webSocketServer *ws.WebSocketServer,
	redisClient database.RedisClient,
) WebSocketService {
	return &webSocketService{
		authContext:     authContext,
		log:             log,
		generator:       generator,
		webSocketServer: webSocketServer,
		redisClient:     redisClient,
	}
}

func (s *webSocketService) OpenWsConnection(ctx context.Context, c *gin.Context, req *entities.OpenWsConnectionRequest) error {
	s.log.InfoWithID(ctx, "[Service: OpenWsConnection] Called")

	authCtx, err := s.authContext.GetAuthContext(ctx)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: OpenWsConnection] Error getting auth context", err)
		return err
	}

	redisClient, err := s.redisClient.Get(ctx, req.SessionToken)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: OpenWsConnection] Error getting session token", err)
		return err
	}

	var sessionPayload entities.RedisSessionToken
	if err := json.Unmarshal([]byte(redisClient), &sessionPayload); err != nil {
		s.log.ErrorWithID(ctx, "[Service: OpenWsConnection] Error unmarshalling session token", err)
		return err
	}

	if sessionPayload.UserID != authCtx.Payload.UserID {
		s.log.ErrorWithID(ctx, "[Service: OpenWsConnection] User ID mismatch", "sessionPayload", sessionPayload, "authCtx", authCtx)
		return err
	}

	s.webSocketServer.HandleConnection(c.Writer, c.Request, sessionPayload)
	return nil
}
