package middleware

import (
	"context"
	"errors"

	"github.com/gin-gonic/gin"
	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
	"gitlab.com/interview-simulation/interview-backend-server/internal/utils"
)

type AuthContext interface {
	ExtractAuthContext(ctx *gin.Context) (context.Context, error)
	GetAuthContext(ctx context.Context) (*AuthPayload, error)
}

type authContext struct {
	log *log.Logger
}

type AuthPayload struct {
	Payload     *utils.SignInTokenPayload
	AccessToken string
}

func NewAuthContext(l *log.Logger) AuthContext {
	return &authContext{
		log: l,
	}
}

func (a *authContext) ExtractAuthContext(ctx *gin.Context) (context.Context, error) {
	a.log.InfoWithID(ctx.Request.Context(), "[Middleware: ExtractAuthContext] Called")
	rawPayload, ok := ctx.Get(string(constants.AuthorizationPayloadKey))
	if !ok {
		a.log.ErrorWithID(ctx.Request.Context(), "[Middleware: ExtractAuthContext] Auth payload not found")
		return ctx.Request.Context(), app_error.New(errors.New("auth payload not found"), app_error.ErrCodeAuthInvalidHeader)
	}

	payload, ok := rawPayload.(*utils.SignInTokenPayload)
	if !ok {
		a.log.ErrorWithID(ctx.Request.Context(), "[Middleware: ExtractAuthContext] Invalid auth payload format")
		return ctx.Request.Context(), app_error.New(errors.New("invalid auth payload format"), app_error.ErrCodeAuthInvalidHeader)
	}

	authCtx := &AuthPayload{
		Payload: payload,
	}

	rawAccessToken, ok := ctx.Get(string(constants.NewAccessTokenKey))
	if ok {
		a.log.InfoWithID(ctx.Request.Context(), "[Middleware: ExtractAuthContext] Access token found")
		accessToken, _ := rawAccessToken.(string)
		authCtx.AccessToken = accessToken
	}

	enrichedCtx := context.WithValue(ctx.Request.Context(), constants.AuthContextKey, authCtx)
	return enrichedCtx, nil
}

func (a *authContext) GetAuthContext(ctx context.Context) (*AuthPayload, error) {
	val := ctx.Value(constants.AuthContextKey)
	a.log.InfoWithID(ctx, "[Middleware: GetAuthContext] Called")

	if val == nil {
		a.log.ErrorWithID(ctx, "[Middleware: GetAuthContext] Missing auth context")
		return nil, app_error.New(errors.New("missing auth context"), app_error.ErrCodeAuthInvalidHeader)
	}
	authCtx, ok := val.(*AuthPayload)
	if !ok {
		a.log.ErrorWithID(ctx, "[Middleware: GetAuthContext] Invalid auth context format")
		return nil, app_error.New(errors.New("invalid auth context format"), app_error.ErrCodeAuthInvalidHeader)
	}
	return authCtx, nil
}
