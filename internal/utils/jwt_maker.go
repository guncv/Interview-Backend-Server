package utils

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"gitlab.com/interview-simulation/interview-backend-server/internal/config"
	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
	"gitlab.com/interview-simulation/interview-backend-server/internal/entities"
	app_error "gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
	"gitlab.com/interview-simulation/interview-backend-server/internal/repositories"
)

type JwtToken interface {
	CreateToken(ctx context.Context, req *entities.TokenRequest) (string, *TokenPayload, error)
	VerifyToken(ctx context.Context, token string) (*TokenPayload, error)
	HashTokenSHA256(ctx context.Context, token string) string
	IsTokenMatch(ctx context.Context, providedToken string, storedTokenHash string) bool
	RenewAccessToken(ctx *gin.Context, token string) (string, *TokenPayload, error)
}

type jwtToken struct {
	config            *config.Config
	logger            *log.Logger
	sessionRepository repositories.SessionRepository
}

func NewJwtToken(
	config *config.Config,
	logger *log.Logger,
	sessionRepository repositories.SessionRepository,
) JwtToken {
	return &jwtToken{
		config:            config,
		logger:            logger,
		sessionRepository: sessionRepository,
	}
}

func (maker *jwtToken) CreateToken(ctx context.Context, req *entities.TokenRequest) (string, *TokenPayload, error) {
	maker.logger.InfoWithID(ctx, "[Utils: CreateToken] Creating token", "req", req)
	payload, err := NewTokenPayload(req)
	if err != nil {
		maker.logger.ErrorWithID(ctx, "[Utils: CreateToken] Error creating token", "error", err)
		return "", nil, app_error.New(err, app_error.ErrCodeGeneralServerUnavailable)
	}

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)
	token, err := jwtToken.SignedString([]byte(maker.config.AuthConfig.JwtSecretKey))
	if err != nil {
		maker.logger.ErrorWithID(ctx, "[Utils: CreateToken] Error creating token", "error", err)
		return "", nil, app_error.New(err, app_error.ErrCodeGeneralServerUnavailable)
	}

	return token, payload, err
}

func (maker *jwtToken) VerifyToken(ctx context.Context, token string) (*TokenPayload, error) {
	maker.logger.InfoWithID(ctx, "[Utils: VerifyToken] Verifying token", "token", token)

	keyFunc := func(token *jwt.Token) (interface{}, error) {
		_, ok := token.Method.(*jwt.SigningMethodHMAC)
		if !ok {
			maker.logger.ErrorWithID(ctx, "[Utils: VerifyToken] Invalid token method", "error", constants.ErrInvalidToken)
			return nil, app_error.New(constants.ErrInvalidToken, app_error.ErrCodeGeneralServerUnavailable)
		}

		return []byte(maker.config.AuthConfig.JwtSecretKey), nil
	}

	jwtToken, err := jwt.ParseWithClaims(token, &TokenPayload{}, keyFunc)
	if err != nil {
		verr, ok := err.(*jwt.ValidationError)
		if ok && errors.Is(verr.Inner, constants.ErrExpiredToken) {
			maker.logger.ErrorWithID(ctx, "[Utils: VerifyToken] Expired token", "error", constants.ErrExpiredToken)
			return nil, app_error.New(constants.ErrExpiredToken, app_error.ErrCodeAuthExpiredToken)
		}
		maker.logger.ErrorWithID(ctx, "[Utils: VerifyToken] Invalid token", "error", constants.ErrInvalidToken)
		return nil, app_error.New(constants.ErrInvalidToken, app_error.ErrCodeAuthInvalidToken)
	}

	payload, ok := jwtToken.Claims.(*TokenPayload)

	if !ok {
		maker.logger.ErrorWithID(ctx, "[Utils: VerifyToken] Invalid token claims", "error", constants.ErrInvalidToken)
		return nil, app_error.New(constants.ErrInvalidToken, app_error.ErrCodeAuthInvalidToken)
	}

	return payload, nil
}

func (maker *jwtToken) HashTokenSHA256(ctx context.Context, token string) string {
	maker.logger.InfoWithID(ctx, "[Utils: HashTokenSHA256] Hashing token", "token", token)
	h := sha256.New()
	h.Write([]byte(token))
	return hex.EncodeToString(h.Sum(nil))
}

func (maker *jwtToken) IsTokenMatch(ctx context.Context, providedToken string, storedTokenHash string) bool {
	maker.logger.InfoWithID(ctx, "[Utils: IsTokenMatch] Checking if token matches", "providedToken", providedToken, "storedTokenHash", storedTokenHash)
	return maker.HashTokenSHA256(ctx, providedToken) == storedTokenHash
}

func (maker *jwtToken) RenewAccessToken(ctx *gin.Context, token string) (string, *TokenPayload, error) {
	maker.logger.InfoWithID(ctx, "[Utils: RenewAccessToken] Renewing access token", "token")

	refreshPayload, err := maker.VerifyToken(ctx, token)
	if err != nil {
		maker.logger.ErrorWithID(ctx, "[Utils: RenewAccessToken] Error verifying token", "error", err)
		return "", nil, app_error.New(err, app_error.ErrCodeAuthInvalidToken)
	}

	session, err := maker.sessionRepository.GetSessionByID(ctx, refreshPayload.ID.String())
	if err != nil {
		maker.logger.ErrorWithID(ctx, "[Utils: RenewAccessToken] Error getting session", "error", err)
		return "", nil, app_error.New(err, app_error.ErrCodeAuthInvalidToken)
	}

	if session.IsRevoked.Bool {
		maker.logger.ErrorWithID(ctx, "[Utils: RenewAccessToken] Session revoked", "error", constants.ErrInvalidToken)
		return "", nil, app_error.New(constants.ErrInvalidToken, app_error.ErrCodeAuthInvalidToken)
	}

	if session.ID.String() != refreshPayload.ID.String() {
		maker.logger.ErrorWithID(ctx, "[Utils: RenewAccessToken] Incorrect session id", "error", constants.ErrInvalidToken)
		return "", nil, app_error.New(constants.ErrInvalidToken, app_error.ErrCodeAuthInvalidToken)
	}

	if session.UserID.String() != refreshPayload.UserID {
		maker.logger.ErrorWithID(ctx, "[Utils: RenewAccessToken] Incorrect session user", "error", constants.ErrInvalidToken)
		return "", nil, app_error.New(constants.ErrInvalidToken, app_error.ErrCodeAuthInvalidToken)
	}

	refreshTokenHash := maker.HashTokenSHA256(ctx, token)

	if session.RefreshTokenHash != refreshTokenHash {
		maker.logger.ErrorWithID(ctx, "[Utils: RenewAccessToken] Mismatch session token", "error", constants.ErrInvalidToken)
		return "", nil, app_error.New(constants.ErrInvalidToken, app_error.ErrCodeAuthInvalidToken)
	}

	if time.Now().After(session.ExpiresAt.Time) {
		maker.logger.ErrorWithID(ctx, "[Utils: RenewAccessToken] Session expired", "error", constants.ErrExpiredToken)
		return "", nil, app_error.New(constants.ErrExpiredToken, app_error.ErrCodeAuthExpiredToken)
	}

	createTokenreq := &entities.TokenRequest{
		UserID:   session.UserID.String(),
		Role:     refreshPayload.Role,
		Duration: maker.config.AuthConfig.AccessTokenDuration,
	}

	accessToken, _, err := maker.CreateToken(ctx, createTokenreq)
	if err != nil {
		return "", nil, err
	}

	return accessToken, refreshPayload, nil
}
