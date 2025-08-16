package utils

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gitlab.com/interview-simulation/interview-backend-server/internal/config"
	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
	"gitlab.com/interview-simulation/interview-backend-server/internal/entities"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
)

type Cookies interface {
	SetCookie(ctx *gin.Context, req *entities.SignInUserByEmailAndPasswordResponse) error
	SetRefreshTokenCookie(c *gin.Context, token string, duration time.Duration, domain string, isRejectHTTP bool)
	ClearRefreshTokenCookie(c *gin.Context)
}

type cookies struct {
	config *config.Config
	log    *log.Logger
}

func NewCookies(config *config.Config, log *log.Logger) Cookies {
	return &cookies{config: config, log: log}
}

func (c *cookies) SetCookie(ctx *gin.Context, req *entities.SignInUserByEmailAndPasswordResponse) error {
	c.log.InfoWithID(ctx, "[Utils: SetCookie] Called")

	c.SetRefreshTokenCookie(ctx, req.RefreshToken,
		c.config.AuthConfig.RefreshTokenDuration,
		c.config.AuthConfig.CookieDomain,
		c.config.AuthConfig.CookieRejectHTTP,
	)

	return nil
}

func (c *cookies) SetRefreshTokenCookie(ctx *gin.Context, token string, duration time.Duration, domain string, isRejectHTTP bool) {
	c.log.InfoWithID(ctx, "[Utils: SetRefreshTokenCookie] Called")

	http.SetCookie(ctx.Writer, &http.Cookie{
		Name:     string(constants.RefreshTokenCookieKey),
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   !isRejectHTTP,
		SameSite: http.SameSiteLaxMode,
		Domain:   domain,
		Expires:  time.Now().Add(duration),
	})
}

func (c *cookies) ClearRefreshTokenCookie(ctx *gin.Context) {
	c.log.InfoWithID(ctx, "[Utils: ClearRefreshTokenCookie] Called")

	http.SetCookie(ctx.Writer, &http.Cookie{
		Name:     string(constants.RefreshTokenCookieKey),
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   !c.config.AuthConfig.CookieRejectHTTP,
		SameSite: http.SameSiteLaxMode,
		Domain:   c.config.AuthConfig.CookieDomain,
		Expires:  time.Now().Add(-time.Hour * 24),
	})
}
