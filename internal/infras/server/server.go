package server

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"gitlab.com/interview-simulation/interview-backend-server/internal/config"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/routes"
	"go.uber.org/dig"
)

type GinServer struct {
	Router    *gin.Engine
	AppConfig *config.AppConfig
}

func (s *GinServer) Start() error {
	addr := fmt.Sprintf(":%s", s.AppConfig.AppPort)
	return s.Router.Run(addr)
}

func NewGinServer(c *config.Config, diContainer *dig.Container) *GinServer {
	router := gin.Default()

	s := &GinServer{
		Router:    router,
		AppConfig: &c.AppConfig,
	}

	routes.RegisterRoutes(router, diContainer)
	return s
}
