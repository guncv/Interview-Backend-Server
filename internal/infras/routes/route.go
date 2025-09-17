package routes

import (
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gitlab.com/interview-simulation/interview-backend-server/internal/config"
	"gitlab.com/interview-simulation/interview-backend-server/internal/handlers"
	"gitlab.com/interview-simulation/interview-backend-server/internal/middleware"
	"go.uber.org/dig"
)

func RegisterRoutes(e *gin.Engine, c *dig.Container, cfg *config.Config) {

	// Use configurable CORS origins
	corsOrigins := cfg.AppConfig.CORSOrigins
	if len(corsOrigins) == 0 {
		// Fallback to default origins if none configured
		corsOrigins = []string{"http://localhost:5173"}
	}

	e.Use(cors.New(cors.Config{
		AllowOrigins:     corsOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Active-Role"},
		ExposeHeaders:    []string{"Content-Length", "Authorization", "Access-Control-Expose-Headers", "X-New-Access-Token"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	e.RedirectTrailingSlash = false
	e.Use(middleware.InjectRequestMetadata())

	e.GET("/api/v1/docs", func(c *gin.Context) {
		c.HTML(http.StatusOK, "swagger.html", gin.H{
			"title": "Interview Simulation API - Swagger UI",
		})
	})

	e.GET("/api/v1/swagger.json", func(c *gin.Context) {
		swaggerBytes, err := os.ReadFile("./docs/swagger.json")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read swagger configuration"})
			return
		}

		// Use the configuration to determine the correct host
		apiHost := cfg.AppConfig.APIHost
		if apiHost == "" {
			apiHost = "localhost:8080" // fallback
		}

		// Replace the placeholder with the actual host
		swaggerContent := strings.ReplaceAll(string(swaggerBytes), "${API_HOST:-localhost:8080}", apiHost)

		c.Header("Content-Type", "application/json")
		c.String(http.StatusOK, swaggerContent)
	})

	if err := c.Invoke(func(
		userHandler *handlers.UserHandler,
		authMiddleware middleware.AuthMiddleware,
		resumeHandler *handlers.ResumeHandler,
		interviewSessionHandler *handlers.InterviewSessionHandler,
	) {
		api_v1 := e.Group("/api/v1")
		userRoutes(api_v1, userHandler, authMiddleware)
		resumeRoutes(api_v1, resumeHandler, authMiddleware)
		websocketRoutes(api_v1, interviewSessionHandler)
		interviewSessionRoutes(api_v1, interviewSessionHandler, authMiddleware)
	}); err != nil {
		panic(err)
	}
}

func userRoutes(eg *gin.RouterGroup, userHandler *handlers.UserHandler, authMiddleware middleware.AuthMiddleware) {
	userRoutes := eg.Group("/auth")
	userMiddleRoutes := eg.Group("/auth").Use(authMiddleware.AuthMiddleware())

	{
		userRoutes.GET("/health", userHandler.HealthCheck)
		userRoutes.POST("/sign-up", userHandler.SignUpUser)
		userRoutes.POST("/verify-email", userHandler.SendVerifyEmail)
		userRoutes.POST("/reset-verify-email", userHandler.ResetVerifyEmailCode)
		userRoutes.POST("/sign-in", userHandler.SignInUserByEmailAndPassword)
		userRoutes.POST("/forgot-password", userHandler.ForgotPassword)
		userRoutes.POST("/reset-password", userHandler.ResetUserPassword)
		userMiddleRoutes.POST("/sign-out", userHandler.SignOut)
	}
}

func resumeRoutes(eg *gin.RouterGroup, resumeHandler *handlers.ResumeHandler, authMiddleware middleware.AuthMiddleware) {
	resumeMiddleRoutes := eg.Group("/resumes").Use(authMiddleware.AuthMiddleware())

	{
		resumeMiddleRoutes.POST("/switch-default", resumeHandler.SwitchDefaultResume)
		resumeMiddleRoutes.GET("/:id", resumeHandler.GetResumeByID)
		resumeMiddleRoutes.GET("/list", resumeHandler.ListResume)
	}
}

func interviewSessionRoutes(eg *gin.RouterGroup, interviewSessionHandler *handlers.InterviewSessionHandler, authMiddleware middleware.AuthMiddleware) {
	interviewSessionMiddleRoutes := eg.Group("/sessions").Use(authMiddleware.AuthMiddleware())

	{
		interviewSessionMiddleRoutes.POST("", interviewSessionHandler.CreateInterviewSessionWithNewResume)
		interviewSessionMiddleRoutes.POST("/existing", interviewSessionHandler.CreateInterviewSessionWithExistingResume)
		interviewSessionMiddleRoutes.GET("/chat-history/:session_token", interviewSessionHandler.GetChatHistoryBySessionToken)
		interviewSessionMiddleRoutes.GET("/information/:session_token", interviewSessionHandler.GetInterviewSessionInformation)
	}
}

func websocketRoutes(eg *gin.RouterGroup, interviewSessionHandler *handlers.InterviewSessionHandler) {
	websocketRoutes := eg.Group("/ws")

	{
		websocketRoutes.GET("/connect/:id", interviewSessionHandler.OpenWsConnection)
	}
}
