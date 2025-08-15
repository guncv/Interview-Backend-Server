package middleware

// import (
// 	"context"
// 	"net/http/httptest"
// 	"testing"

// 	"github.com/gin-gonic/gin"
// 	"github.com/stretchr/testify/assert"
// 	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
// )

// func TestContext_InjectRequestMetadata(t *testing.T) {
// 	testCases := []struct {
// 		name   string
// 		setup  func() *gin.Context
// 		verify func(t *testing.T, ctx context.Context, gotErr error)
// 	}{
// 		{
// 			name: "InjectRequestMetadata_OK",
// 			setup: func() *gin.Context {
// 				w := httptest.NewRecorder()
// 				req := httptest.NewRequest("GET", "/test", nil)
// 				req.Header.Set("User-Agent", "user-agent")
// 				req.Header.Set("X-Forwarded-For", "127.0.0.1")

// 				c, _ := gin.CreateTestContext(w)
// 				c.Request = req
// 				return c
// 			},
// 			verify: func(t *testing.T, ctx context.Context, gotErr error) {
// 				assert.NoError(t, gotErr)
// 				assert.NotNil(t, ctx)

// 				assert.Equal(t, "user-agent", ctx.Value(constants.UserAgentKey))
// 				assert.Equal(t, "127.0.0.1", ctx.Value(constants.ClientIPKey))
// 			},
// 		},
// 		{
// 			name: "InjectRequestMetadata_MissingUserAgent",
// 			setup: func() *gin.Context {
// 				w := httptest.NewRecorder()
// 				req := httptest.NewRequest("GET", "/test", nil)
// 				req.Header.Set("X-Forwarded-For", "127.0.0.1")

// 				c, _ := gin.CreateTestContext(w)
// 				c.Request = req
// 				return c
// 			},
// 			verify: func(t *testing.T, ctx context.Context, gotErr error) {
// 				assert.NoError(t, gotErr)
// 				assert.NotNil(t, ctx)

// 				assert.Empty(t, ctx.Value(constants.UserAgentKey))
// 				assert.Equal(t, "127.0.0.1", ctx.Value(constants.ClientIPKey))
// 			},
// 		},
// 		{
// 			name: "InjectRequestMetadata_MissingClientIP",
// 			setup: func() *gin.Context {
// 				w := httptest.NewRecorder()
// 				req := httptest.NewRequest("GET", "/test", nil)
// 				req.Header.Set("User-Agent", "user-agent")

// 				c, _ := gin.CreateTestContext(w)
// 				c.Request = req
// 				return c
// 			},
// 			verify: func(t *testing.T, ctx context.Context, gotErr error) {
// 				assert.NoError(t, gotErr)
// 				assert.NotNil(t, ctx)

// 				assert.Equal(t, "user-agent", ctx.Value(constants.UserAgentKey))
// 				// Gin's ClientIP() returns a default IP when no headers are present
// 				assert.Equal(t, "192.0.2.1", ctx.Value(constants.ClientIPKey))
// 			},
// 		},
// 	}

// 	for _, tC := range testCases {
// 		t.Run(tC.name, func(t *testing.T) {
// 			c := tC.setup()

// 			InjectRequestMetadata()(c)

// 			tC.verify(t, c.Request.Context(), nil)
// 		})
// 	}
// }
