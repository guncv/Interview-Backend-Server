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
	db "gitlab.com/interview-simulation/interview-backend-server/internal/db/sqlc"
	"gitlab.com/interview-simulation/interview-backend-server/internal/entities"
	app_error "gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
	"gitlab.com/interview-simulation/interview-backend-server/internal/repositories"
)

type JwtToken interface {
	CreateWebSocketSessionToken(ctx context.Context, req *entities.WebSocketSessionReq) (string, error)
	CreateToken(ctx context.Context, req *entities.TokenRequest) (string, *SignInTokenPayload, error)
	VerifyToken(ctx context.Context, token string, secretKey string) (*SignInTokenPayload, error)
	CreateJWTToken(ctx context.Context, claims jwt.Claims, secretKey string) (string, error)
	HashTokenSHA256(ctx context.Context, token string) string
	IsTokenMatch(ctx context.Context, providedToken string, storedTokenHash string) bool
	RenewAccessToken(ctx *gin.Context, token string) (string, *SignInTokenPayload, error)
}

type jwtToken struct {
	config            *config.Config
	logger            *log.Logger
	sessionRepository repositories.AuthSessionRepository
}

func NewJwtToken(
	config *config.Config,
	logger *log.Logger,
	sessionRepository repositories.AuthSessionRepository,
) JwtToken {
	return &jwtToken{
		config:            config,
		logger:            logger,
		sessionRepository: sessionRepository,
	}
}

func (maker *jwtToken) CreateJWTToken(ctx context.Context, claims jwt.Claims, secretKey string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(secretKey))
	if err != nil {
		maker.logger.ErrorWithID(ctx, "[Utils: JWT] Error signing token", "error", err)
		return "", app_error.New(err, app_error.ErrCodeGeneralServerUnavailable)
	}
	return signedToken, nil
}

func (maker *jwtToken) CreateWebSocketSessionToken(ctx context.Context, req *entities.WebSocketSessionReq) (string, error) {

	payload := NewWebSocketSessionPayload(req)

	token, err := maker.CreateJWTToken(ctx, payload, maker.config.InterviewSessionConfig.EncryptionSecretKey)
	if err != nil {
		return "", err
	}

	return token, nil
}

func (maker *jwtToken) CreateToken(ctx context.Context, req *entities.TokenRequest) (string, *SignInTokenPayload, error) {

	payload, err := NewSignInTokenPayload(req)
	if err != nil {
		maker.logger.ErrorWithID(ctx, "[Utils: JWT] Error creating sign-in token payload", "error", err)
		return "", nil, app_error.New(err, app_error.ErrCodeGeneralServerUnavailable)
	}

	token, err := maker.CreateJWTToken(ctx, payload, maker.config.AuthConfig.EncryptionSecretKey)
	if err != nil {
		maker.logger.ErrorWithID(ctx, "[Utils: JWT] Error creating sign-in token", "error", err)
		return "", nil, err
	}

	return token, payload, nil
}

func (maker *jwtToken) VerifyToken(ctx context.Context, token string, secretKey string) (*SignInTokenPayload, error) {

	payload := &SignInTokenPayload{}
	if err := maker.verifyJWTToken(ctx, token, payload, secretKey); err != nil {
		maker.logger.ErrorWithID(ctx, "[Utils: JWT] Error verifying sign-in token", "error", err)
		return nil, err
	}

	return payload, nil
}

func (maker *jwtToken) verifyJWTToken(ctx context.Context, tokenString string, claims jwt.Claims, secretKey string) error {
	keyFunc := func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			maker.logger.ErrorWithID(ctx, "[Utils: JWT] Invalid token method", "error", constants.ErrInvalidToken)
			return nil, errors.New("invalid token method")
		}
		return []byte(secretKey), nil
	}

	parser := jwt.Parser{
		SkipClaimsValidation: true,
	}

	token, err := parser.ParseWithClaims(tokenString, claims, keyFunc)
	if err != nil {
		maker.logger.ErrorWithID(ctx, "[Utils: JWT] Invalid token signature", "error", constants.ErrInvalidToken)
		return app_error.New(constants.ErrInvalidToken, app_error.ErrCodeAuthInvalidToken)
	}

	if err := maker.validateClaims(ctx, token.Claims); err != nil {
		return err
	}

	return nil
}

func (maker *jwtToken) validateClaims(ctx context.Context, targetClaims jwt.Claims) error {
	if payload, ok := targetClaims.(interface{ GetExpiredAt() time.Time }); ok {
		expiredAt := payload.GetExpiredAt()
		now := time.Now()

		if now.After(expiredAt) {
			graceWindow := maker.config.AuthConfig.TokenGraceWindow
			if graceWindow == 0 {
				graceWindow = time.Minute
			}

			if time.Since(expiredAt) > graceWindow {
				maker.logger.ErrorWithID(ctx, "[Utils: JWT] Expired token outside grace window",
					"error", constants.ErrExpiredToken, "expiredAt", expiredAt, "graceWindow", graceWindow)
				return app_error.New(constants.ErrExpiredToken, app_error.ErrCodeAuthExpiredToken)
			}
		}
	}

	return nil
}

func (maker *jwtToken) HashTokenSHA256(ctx context.Context, token string) string {
	h := sha256.New()
	h.Write([]byte(token))
	return hex.EncodeToString(h.Sum(nil))
}

func (maker *jwtToken) IsTokenMatch(ctx context.Context, providedToken string, storedTokenHash string) bool {
	return maker.HashTokenSHA256(ctx, providedToken) == storedTokenHash
}

func (maker *jwtToken) RenewAccessToken(ctx *gin.Context, token string) (string, *SignInTokenPayload, error) {

	refreshPayload, err := maker.VerifyToken(ctx, token, maker.config.AuthConfig.EncryptionSecretKey)
	if err != nil {
		return "", nil, err
	}

	session, err := maker.checkSessionByID(ctx, refreshPayload)
	if err != nil {
		maker.logger.ErrorWithID(ctx, "[Utils: RenewAccessToken] Error checking session by id", "error", err)
		return "", nil, err
	}

	if err = maker.isRefreshTokenValidWithSession(ctx, token, session); err != nil {
		maker.logger.ErrorWithID(ctx, "[Utils: RenewAccessToken] Error checking refresh token with session", "error", err)
		return "", nil, err
	}

	createTokenreq := &entities.TokenRequest{
		UserID:   session.UserID.String(),
		Role:     refreshPayload.Role,
		Duration: maker.config.AuthConfig.AccessTokenDuration,
	}

	accessToken, _, err := maker.CreateToken(ctx, createTokenreq)
	if err != nil {
		maker.logger.ErrorWithID(ctx, "[Utils: RenewAccessToken] Error creating token", "error", err)
		return "", nil, err
	}

	return accessToken, refreshPayload, nil
}

func (maker *jwtToken) checkSessionByID(ctx context.Context, refreshPayload *SignInTokenPayload) (*db.AuthSessions, error) {

	session, err := maker.sessionRepository.GetAuthSessionByID(ctx, refreshPayload.ID)
	if err != nil {
		maker.logger.ErrorWithID(ctx, "[Utils: checkSessionByID] Error getting session", "error", err)
		return nil, app_error.New(err, app_error.ErrCodeAuthInvalidToken)
	}

	if session.IsRevoked.Bool {
		maker.logger.ErrorWithID(ctx, "[Utils: checkSessionByID] Session revoked", "error", constants.ErrInvalidToken)
		return nil, app_error.New(constants.ErrInvalidToken, app_error.ErrCodeAuthInvalidToken)
	}

	if session.ID.String() != refreshPayload.ID.String() {
		maker.logger.ErrorWithID(ctx, "[Utils: checkSessionByID] Incorrect session id", "error", constants.ErrInvalidToken)
		return nil, app_error.New(constants.ErrInvalidToken, app_error.ErrCodeAuthInvalidToken)
	}

	if session.UserID.String() != refreshPayload.UserID {
		maker.logger.ErrorWithID(ctx, "[Utils: checkSessionByID] Incorrect session user", "error", constants.ErrInvalidToken)
		return nil, app_error.New(constants.ErrInvalidToken, app_error.ErrCodeAuthInvalidToken)
	}

	return session, nil
}

func (maker *jwtToken) isRefreshTokenValidWithSession(ctx context.Context, token string, session *db.AuthSessions) error {
	refreshTokenHash := maker.HashTokenSHA256(ctx, token)

	if session.RefreshTokenHash != refreshTokenHash {
		maker.logger.ErrorWithID(ctx, "[Utils: isRefreshTokenValidWithSession] Mismatch session token", "error", constants.ErrInvalidToken)
		return app_error.New(constants.ErrInvalidToken, app_error.ErrCodeAuthInvalidToken)
	}

	if time.Now().After(session.ExpiresAt.Time) {
		maker.logger.ErrorWithID(ctx, "[Utils: isRefreshTokenValidWithSession] Session expired", "error", constants.ErrExpiredToken)
		return app_error.New(constants.ErrExpiredToken, app_error.ErrCodeAuthExpiredToken)
	}

	return nil
}
