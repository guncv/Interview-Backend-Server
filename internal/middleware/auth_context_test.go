package middleware

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
	"gitlab.com/interview-simulation/interview-backend-server/internal/utils"
)

func TestAuthContext_ExtractAuthContext(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)

	testCases := []struct {
		name   string
		setup  func() *gin.Context
		verify func(t *testing.T, ctx context.Context, gotErr error)
	}{
		{
			name: "ExtractAuthContext_OK",
			setup: func() *gin.Context {
				w := httptest.NewRecorder()
				req := httptest.NewRequest("GET", "/test", nil)
				c, _ := gin.CreateTestContext(w)
				c.Request = req
				c.Set(string(constants.AuthorizationPayloadKey), &utils.SignInTokenPayload{
					UserID: "user123",
					Role:   constants.UserRoleAdmin,
				})
				c.Set(string(constants.NewAccessTokenKey), "accessToken")
				return c
			},
			verify: func(t *testing.T, ctx context.Context, gotErr error) {
				assert.NotNil(t, ctx)
				assert.NoError(t, gotErr)

				authPayload := ctx.Value(constants.AuthContextKey).(*AuthPayload)
				assert.NotNil(t, authPayload)
				assert.Equal(t, "user123", authPayload.Payload.UserID)
				assert.Equal(t, constants.UserRoleAdmin, authPayload.Payload.Role)
				assert.Equal(t, "accessToken", authPayload.AccessToken)
			},
		},
		{
			name: "ExtractAuthContext_PayloadNotFound",
			setup: func() *gin.Context {
				w := httptest.NewRecorder()
				req := httptest.NewRequest("GET", "/test", nil)
				c, _ := gin.CreateTestContext(w)
				c.Request = req

				c.Set(string(constants.NewAccessTokenKey), "accessToken")
				return c
			},
			verify: func(t *testing.T, ctx context.Context, gotErr error) {
				assert.Error(t, gotErr)
				assert.Empty(t, ctx)
			},
		},
		{
			name: "ExtractAuthContext_AccessTokenNotFound",
			setup: func() *gin.Context {
				w := httptest.NewRecorder()
				req := httptest.NewRequest("GET", "/test", nil)
				c, _ := gin.CreateTestContext(w)
				c.Request = req
				c.Set(string(constants.AuthorizationPayloadKey), &utils.SignInTokenPayload{
					UserID: "user123",
					Role:   constants.UserRoleAdmin,
				})
				return c
			},
			verify: func(t *testing.T, ctx context.Context, gotErr error) {
				assert.NotNil(t, ctx)
				assert.NoError(t, gotErr)

				authPayload := ctx.Value(constants.AuthContextKey).(*AuthPayload)
				assert.NotNil(t, authPayload)
				assert.Equal(t, "user123", authPayload.Payload.UserID)
				assert.Equal(t, constants.UserRoleAdmin, authPayload.Payload.Role)
				assert.Empty(t, authPayload.AccessToken)
			},
		},
		{
			name: "ExtractAuthContext_InvalidPayloadFormat",
			setup: func() *gin.Context {
				w := httptest.NewRecorder()
				req := httptest.NewRequest("GET", "/test", nil)
				c, _ := gin.CreateTestContext(w)
				c.Request = req
				c.Set(string(constants.AuthorizationPayloadKey), "invalidPayload")
				c.Set(string(constants.NewAccessTokenKey), "accessToken")
				return c
			},
			verify: func(t *testing.T, ctx context.Context, gotErr error) {
				assert.Error(t, gotErr)
				assert.Empty(t, ctx)
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			c := tC.setup()
			svc := NewAuthContext(lgr)
			got, gotErr := svc.ExtractAuthContext(c)
			gin.SetMode(gin.TestMode)

			tC.verify(t, got, gotErr)
		})
	}
}

func TestAuthContext_GetAuthContext(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)

	testCases := []struct {
		name   string
		setup  func() context.Context
		verify func(t *testing.T, ctx *AuthPayload, gotErr error)
	}{
		{
			name: "GetAuthContext_OK",
			setup: func() context.Context {
				payload := &utils.SignInTokenPayload{
					UserID: "user123",
					Role:   constants.UserRoleAdmin,
				}

				ctx := context.WithValue(context.Background(), constants.AuthContextKey, &AuthPayload{
					Payload:     payload,
					AccessToken: "accessToken",
				})
				return ctx
			},
			verify: func(t *testing.T, ctx *AuthPayload, gotErr error) {
				assert.NoError(t, gotErr)
				assert.NotNil(t, ctx)

				assert.Equal(t, "user123", ctx.Payload.UserID)
				assert.Equal(t, constants.UserRoleAdmin, ctx.Payload.Role)
				assert.Equal(t, "accessToken", ctx.AccessToken)
			},
		},
		{
			name: "GetAuthContext_MissingPayload",
			setup: func() context.Context {
				ctx := context.Background()
				return ctx
			},
			verify: func(t *testing.T, ctx *AuthPayload, gotErr error) {
				assert.Error(t, gotErr)
				assert.Empty(t, ctx)
			},
		},
		{
			name: "GetAuthContext_InvalidPayloadFormat",
			setup: func() context.Context {
				ctx := context.WithValue(context.Background(), constants.AuthContextKey, "Hello")
				return ctx
			},
			verify: func(t *testing.T, ctx *AuthPayload, gotErr error) {
				assert.Error(t, gotErr)
				assert.Empty(t, ctx)
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			c := tC.setup()
			svc := NewAuthContext(lgr)
			got, gotErr := svc.GetAuthContext(c)

			tC.verify(t, got, gotErr)
		})
	}
}
