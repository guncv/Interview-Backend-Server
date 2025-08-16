package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
	"gitlab.com/interview-simulation/interview-backend-server/internal/utils"
)

type AuthMiddleware interface {
	AuthMiddleware() gin.HandlerFunc
}

type authMiddleware struct {
	tokenMaker utils.JwtToken
	log        *log.Logger
}

func NewAuthMiddleware(tokenMaker utils.JwtToken, log *log.Logger) AuthMiddleware {
	return &authMiddleware{
		tokenMaker: tokenMaker,
		log:        log,
	}
}

func (m *authMiddleware) AuthMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		m.log.InfoWithID(ctx.Request.Context(), "[Middleware: AuthMiddleware] Called")

		authorizationHeader := ctx.GetHeader(string(constants.AuthorizationHeaderKey))
		if len(authorizationHeader) == 0 {
			err := app_error.New(errors.New("authorization header is not provided"), app_error.ErrCodeAuthInvalidHeader)
			m.log.ErrorWithID(ctx.Request.Context(), "[Middleware: AuthMiddleware] Error", err)
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, app_error.New(err, app_error.ErrCodeAuthInvalidHeader))
			return
		}

		fields := strings.Fields(authorizationHeader)
		if len(fields) < 2 {
			err := app_error.New(errors.New("invalid authorization header format"), app_error.ErrCodeAuthInvalidHeader)
			m.log.ErrorWithID(ctx.Request.Context(), "[Middleware: AuthMiddleware] Error", err)
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, app_error.New(err, app_error.ErrCodeAuthInvalidHeader))
			return
		}

		authorizationType := strings.ToLower(fields[0])
		if authorizationType != string(constants.AuthorizationTypeBearer) {
			err := app_error.New(errors.New("authorization header must start with "+string(constants.AuthorizationTypeBearer)), app_error.ErrCodeAuthInvalidHeader)
			m.log.ErrorWithID(ctx.Request.Context(), "[Middleware: AuthMiddleware] Error", err)
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, app_error.New(err, app_error.ErrCodeAuthInvalidHeader))
			return
		}

		accessToken := fields[1]
		payload, err := m.tokenMaker.VerifyToken(ctx.Request.Context(), accessToken)
		if err != nil {
			var appErr *app_error.AppError
			if errors.As(err, &appErr) && appErr.Code == app_error.ErrCodeAuthExpiredToken {
				m.log.ErrorWithID(ctx.Request.Context(), "[Middleware: AuthMiddleware] Error", err)
				ctx.AbortWithStatusJSON(http.StatusUnauthorized, app_error.New(err, app_error.ErrCodeAuthExpiredToken))
				return
			}

			m.log.ErrorWithID(ctx.Request.Context(), "[Middleware: AuthMiddleware] Verify access token error", err)
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, app_error.New(err, app_error.ErrCodeAuthInvalidToken))
			return
		}

		m.log.InfoWithID(ctx.Request.Context(), "[Middleware: AuthMiddleware] Successfully verified token", "payload", payload)
		ctx.Set(string(constants.AuthorizationPayloadKey), payload)
		ctx.Next()

	}
}
