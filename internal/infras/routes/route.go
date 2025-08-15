package routes

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gitlab.com/interview-simulation/interview-backend-server/internal/handlers"
	"gitlab.com/interview-simulation/interview-backend-server/internal/middleware"
	"go.uber.org/dig"
)

func RegisterRoutes(e *gin.Engine, c *dig.Container) {
	e.Use(middleware.InjectRequestMetadata())
	e.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Active-Role"},
		ExposeHeaders:    []string{"Content-Length", "Authorization", "Access-Control-Expose-Headers", "X-New-Access-Token"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	if err := c.Invoke(func(
		userHandler *handlers.UserHandler,
		authMiddleware middleware.AuthMiddleware,
	) {
		api_v1 := e.Group("/api/v1")
		userRoutes(api_v1, userHandler, authMiddleware)
	}); err != nil {
		panic(err)
	}
}

func userRoutes(eg *gin.RouterGroup, userHandler *handlers.UserHandler, authMiddleware middleware.AuthMiddleware) {
	userRoutes := eg.Group("/user")

	{
		userRoutes.GET("/health", userHandler.HealthCheck)
		userRoutes.POST("/sign-up", userHandler.SignUpUser)
	}
}
