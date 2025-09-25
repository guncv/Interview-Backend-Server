package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gitlab.com/interview-simulation/interview-backend-server/internal/config"
	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
	"gitlab.com/interview-simulation/interview-backend-server/internal/utils"
)

type AuthMiddleware interface {
	AuthMiddleware() gin.HandlerFunc
	VerifyAndRenewAccessToken(ctx *gin.Context, accessToken string) (*utils.SignInTokenPayload, error)
}

type authMiddleware struct {
	tokenMaker utils.JwtToken
	log        *log.Logger
	cfg        *config.Config
}

func NewAuthMiddleware(tokenMaker utils.JwtToken, log *log.Logger, cfg *config.Config) AuthMiddleware {
	return &authMiddleware{
		tokenMaker: tokenMaker,
		log:        log,
		cfg:        cfg,
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
		payload, err := m.VerifyAndRenewAccessToken(ctx, accessToken)
		if err != nil {
			m.log.ErrorWithID(ctx.Request.Context(), "[Middleware: AuthMiddleware] Error", err)
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, err)
			return
		}

		ctx.Set(string(constants.AuthorizationPayloadKey), payload)
		ctx.Next()

	}
}

func (m *authMiddleware) VerifyAndRenewAccessToken(ctx *gin.Context, accessToken string) (*utils.SignInTokenPayload, error) {
	m.log.InfoWithID(ctx.Request.Context(), "[Middleware: AuthMiddleware] Verify and renew access token", "accessToken", accessToken)

	payload, err := m.tokenMaker.VerifyToken(ctx.Request.Context(), accessToken, m.cfg.AuthConfig.EncryptionSecretKey)
	if err != nil {
		var appErr *app_error.AppError
		if errors.As(err, &appErr) && appErr.Code == app_error.ErrCodeAuthExpiredToken {
			m.log.InfoWithID(ctx.Request.Context(), "[Middleware: AuthMiddleware] Access token expired")

			cookie, err := ctx.Request.Cookie("refresh_token")
			if err != nil {
				m.log.ErrorWithID(ctx.Request.Context(), "[Middleware: AuthMiddleware] Get refresh token error", err)
				return nil, app_error.New(err, app_error.ErrCodeAuthExpiredToken)
			}

			m.log.InfoWithID(ctx.Request.Context(), "[Middleware: AuthMiddleware] Renewing access token", "refreshToken", cookie.Value)
			refreshToken := cookie.Value
			accessToken, payload, err := m.tokenMaker.RenewAccessToken(ctx, refreshToken)
			if err != nil {
				m.log.ErrorWithID(ctx.Request.Context(), "[Middleware: AuthMiddleware] Renew access token error", err)
				return nil, app_error.New(err, app_error.ErrCodeAuthExpiredToken)
			}
			ctx.Set(string(constants.AuthorizationPayloadKey), payload)
			ctx.Set(string(constants.NewAccessTokenKey), accessToken)

			ctx.Header(string(constants.XAccessTokenHeaderKey), accessToken)

			ctx.Next()
			return payload, nil
		}

		m.log.ErrorWithID(ctx.Request.Context(), "[Middleware: AuthMiddleware] Verify access token error", err)
		return nil, app_error.New(err, app_error.ErrCodeAuthInvalidToken)
	}

	return payload, nil
}
