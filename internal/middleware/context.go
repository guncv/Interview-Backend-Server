package middleware

import (
	"context"

	"github.com/gin-gonic/gin"
	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
)

func InjectRequestMetadata() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := context.WithValue(c.Request.Context(), constants.UserAgentKey, c.Request.UserAgent())
		ctx = context.WithValue(ctx, constants.ClientIPKey, c.ClientIP())
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
