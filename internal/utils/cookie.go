package utils

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gitlab.com/interview-simulation/interview-backend-server/internal/config"
	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
)

type Cookies interface {
	SetRefreshTokenCookie(c *gin.Context, refreshToken string)
	ClearRefreshTokenCookie(c *gin.Context)
}

type cookies struct {
	config *config.Config
	log    *log.Logger
}

func NewCookies(config *config.Config, log *log.Logger) Cookies {
	return &cookies{config: config, log: log}
}

func (c *cookies) SetRefreshTokenCookie(ctx *gin.Context, refreshToken string) {
	c.log.InfoWithID(ctx, "[Utils: SetRefreshTokenCookie] Called")

	duration := c.config.AuthConfig.RefreshTokenDuration
	domain := c.config.AuthConfig.CookieDomain
	isRejectHTTP := c.config.AuthConfig.CookieRejectHTTP
	expiration := time.Now().Add(duration)
	secure := !isRejectHTTP
	c.setCookie(ctx, refreshToken, expiration, domain, secure)
}

func (c *cookies) ClearRefreshTokenCookie(ctx *gin.Context) {
	c.log.InfoWithID(ctx, "[Utils: ClearRefreshTokenCookie] Called")

	expiration := time.Now().Add(-24 * time.Hour)
	secure := !c.config.AuthConfig.CookieRejectHTTP
	domain := c.config.AuthConfig.CookieDomain
	c.setCookie(ctx, "", expiration, domain, secure)
}

func (c *cookies) setCookie(ctx *gin.Context, value string, expires time.Time, domain string, secure bool) {
	http.SetCookie(ctx.Writer, &http.Cookie{
		Name:     string(constants.RefreshTokenCookieKey),
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		Domain:   domain,
		Expires:  expires,
	})
}
