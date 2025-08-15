package utils

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gitlab.com/interview-simulation/interview-backend-server/internal/config"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
)

type Cookies interface {
	SetRefreshTokenCookie(c *gin.Context, token string, duration time.Duration, domain string, isRejectHTTP bool)
}

type cookies struct {
	config *config.Config
	log    *log.Logger
}

func NewCookies(config *config.Config, log *log.Logger) Cookies {
	return &cookies{config: config, log: log}
}

// func (c *cookies) SetCookie(ctx *gin.Context, req *entities.SignInServiceResponse) (*entities.SignInResponse, error) {
// 	c.log.InfoWithID(ctx, "[Utils: SetCookie] Called")

// 	c.SetRefreshTokenCookie(ctx, req.RefreshToken,
// 		c.config.AuthConfig.RefreshTokenDuration,
// 		c.config.AuthConfig.CookieDomain,
// 		c.config.AuthConfig.CookieRejectHTTP,
// 	)

// 	resp := entities.SignInResponse{
// 		ID:             req.ID,
// 		AccessToken:    req.AccessToken,
// 		IsTempPassword: req.IsTempPassword,
// 	}

// 	return &resp, nil
// }

func (c *cookies) SetRefreshTokenCookie(ctx *gin.Context, token string, duration time.Duration, domain string, isRejectHTTP bool) {
	c.log.InfoWithID(ctx, "[Utils: SetRefreshTokenCookie] Called")

	http.SetCookie(ctx.Writer, &http.Cookie{
		Name:     "refresh_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   !isRejectHTTP,
		SameSite: http.SameSiteLaxMode,
		Domain:   domain,
		Expires:  time.Now().Add(duration),
	})
}
