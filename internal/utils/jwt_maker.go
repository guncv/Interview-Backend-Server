package utils

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/google/uuid"
	"gitlab.com/interview-simulation/interview-backend-server/internal/config"
	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
	db "gitlab.com/interview-simulation/interview-backend-server/internal/db/sqlc"
	"gitlab.com/interview-simulation/interview-backend-server/internal/entities"
	app_error "gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
	"gitlab.com/interview-simulation/interview-backend-server/internal/repositories"
)

type JwtToken interface {
	CreateVerifyEmailToken(ctx context.Context, req *entities.VerifyEmailTokenRequest) (string, *VerifyEmailTokenPayload, error)
	VerifyVerifyEmailToken(ctx context.Context, token string) (*VerifyEmailTokenPayload, error)
	CreateToken(ctx context.Context, req *entities.TokenRequest) (string, *SignInTokenPayload, error)
	VerifyToken(ctx context.Context, token string) (*SignInTokenPayload, error)
	HashTokenSHA256(ctx context.Context, token string) string
	IsTokenMatch(ctx context.Context, providedToken string, storedTokenHash string) bool
	RenewVerifyEmailToken(ctx context.Context, oldToken string) (string, *VerifyEmailTokenPayload, error)
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

func (maker *jwtToken) createJWTToken(ctx context.Context, claims jwt.Claims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(maker.config.AuthConfig.JwtSecretKey))
	if err != nil {
		maker.logger.ErrorWithID(ctx, "[Utils: JWT] Error signing token", "error", err)
		return "", app_error.New(err, app_error.ErrCodeGeneralServerUnavailable)
	}
	return signedToken, nil
}

func (maker *jwtToken) verifyJWTToken(ctx context.Context, tokenString string, claims jwt.Claims) error {
	keyFunc := func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			maker.logger.ErrorWithID(ctx, "[Utils: JWT] Invalid token method", "error", constants.ErrInvalidToken)
			return nil, app_error.New(constants.ErrInvalidToken, app_error.ErrCodeAuthInvalidToken)
		}
		return []byte(maker.config.AuthConfig.JwtSecretKey), nil
	}

	token, err := jwt.ParseWithClaims(tokenString, claims, keyFunc)
	if err != nil {
		if verr, ok := err.(*jwt.ValidationError); ok && errors.Is(verr.Inner, constants.ErrExpiredToken) {
			maker.logger.ErrorWithID(ctx, "[Utils: JWT] Expired token", "error", constants.ErrExpiredToken)
			return app_error.New(constants.ErrExpiredToken, app_error.ErrCodeAuthExpiredToken)
		}
		maker.logger.ErrorWithID(ctx, "[Utils: JWT] Invalid token", "error", constants.ErrInvalidToken)
		return app_error.New(constants.ErrInvalidToken, app_error.ErrCodeAuthInvalidToken)
	}

	if !token.Valid {
		maker.logger.ErrorWithID(ctx, "[Utils: JWT] Invalid token claims", "error", constants.ErrInvalidToken)
		return app_error.New(constants.ErrInvalidToken, app_error.ErrCodeAuthInvalidToken)
	}

	return nil
}

func (maker *jwtToken) CreateVerifyEmailToken(ctx context.Context, req *entities.VerifyEmailTokenRequest) (string, *VerifyEmailTokenPayload, error) {
	maker.logger.InfoWithID(ctx, "[Utils: JWT] Creating verify email token", "req", req)

	payload, err := NewVerifyEmailTokenPayload(req)
	if err != nil {
		maker.logger.ErrorWithID(ctx, "[Utils: JWT] Error creating verify email token payload", "error", err)
		return "", nil, app_error.New(err, app_error.ErrCodeGeneralServerUnavailable)
	}

	token, err := maker.createJWTToken(ctx, payload)
	if err != nil {
		return "", nil, err
	}

	return token, payload, nil
}

func (maker *jwtToken) VerifyVerifyEmailToken(ctx context.Context, token string) (*VerifyEmailTokenPayload, error) {
	maker.logger.InfoWithID(ctx, "[Utils: JWT] Verifying verify email token", "token", token)

	payload := &VerifyEmailTokenPayload{}
	if err := maker.verifyJWTToken(ctx, token, payload); err != nil {
		return nil, err
	}

	return payload, nil
}

func (maker *jwtToken) CreateToken(ctx context.Context, req *entities.TokenRequest) (string, *SignInTokenPayload, error) {
	maker.logger.InfoWithID(ctx, "[Utils: JWT] Creating sign-in token", "req", req)

	payload, err := NewSignInTokenPayload(req)
	if err != nil {
		maker.logger.ErrorWithID(ctx, "[Utils: JWT] Error creating sign-in token payload", "error", err)
		return "", nil, app_error.New(err, app_error.ErrCodeGeneralServerUnavailable)
	}

	token, err := maker.createJWTToken(ctx, payload)
	if err != nil {
		return "", nil, err
	}

	return token, payload, nil
}

func (maker *jwtToken) VerifyToken(ctx context.Context, token string) (*SignInTokenPayload, error) {
	maker.logger.InfoWithID(ctx, "[Utils: JWT] Verifying sign-in token", "token", token)

	payload := &SignInTokenPayload{}
	if err := maker.verifyJWTToken(ctx, token, payload); err != nil {
		return nil, err
	}

	return payload, nil
}

func (maker *jwtToken) HashTokenSHA256(ctx context.Context, token string) string {
	maker.logger.InfoWithID(ctx, "[Utils: JWT] Hashing token", "token", token)
	h := sha256.New()
	h.Write([]byte(token))
	return hex.EncodeToString(h.Sum(nil))
}

func (maker *jwtToken) IsTokenMatch(ctx context.Context, providedToken string, storedTokenHash string) bool {
	maker.logger.InfoWithID(ctx, "[Utils: JWT] Checking if token matches", "providedToken", providedToken, "storedTokenHash", storedTokenHash)
	return maker.HashTokenSHA256(ctx, providedToken) == storedTokenHash
}

func (maker *jwtToken) validateSession(ctx context.Context, session db.Sessions, refreshPayload *SignInTokenPayload, token string) error {
	if session.IsRevoked.Bool {
		maker.logger.ErrorWithID(ctx, "[Utils: JWT] Session revoked", "error", constants.ErrInvalidToken)
		return app_error.New(constants.ErrInvalidToken, app_error.ErrCodeAuthInvalidToken)
	}

	if session.ID.String() != refreshPayload.ID.String() {
		maker.logger.ErrorWithID(ctx, "[Utils: JWT] Incorrect session id", "error", constants.ErrInvalidToken)
		return app_error.New(constants.ErrInvalidToken, app_error.ErrCodeAuthInvalidToken)
	}

	if session.UserID.String() != refreshPayload.UserID {
		maker.logger.ErrorWithID(ctx, "[Utils: JWT] Incorrect session user", "error", constants.ErrInvalidToken)
		return app_error.New(constants.ErrInvalidToken, app_error.ErrCodeAuthInvalidToken)
	}

	refreshTokenHash := maker.HashTokenSHA256(ctx, token)
	if session.RefreshTokenHash != refreshTokenHash {
		maker.logger.ErrorWithID(ctx, "[Utils: JWT] Mismatch session token", "error", constants.ErrInvalidToken)
		return app_error.New(constants.ErrInvalidToken, app_error.ErrCodeAuthInvalidToken)
	}

	if time.Now().After(session.ExpiresAt.Time) {
		maker.logger.ErrorWithID(ctx, "[Utils: JWT] Session expired", "error", constants.ErrExpiredToken)
		return app_error.New(constants.ErrExpiredToken, app_error.ErrCodeAuthExpiredToken)
	}

	return nil
}

func (maker *jwtToken) RenewVerifyEmailToken(ctx context.Context, oldToken string) (string, *VerifyEmailTokenPayload, error) {
	maker.logger.InfoWithID(ctx, "[Utils: JWT] Renewing verify email token", "oldToken", oldToken)

	keyFunc := func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			maker.logger.ErrorWithID(ctx, "[Utils: JWT] Invalid token method", "error", constants.ErrInvalidToken)
			return nil, app_error.New(constants.ErrInvalidToken, app_error.ErrCodeAuthInvalidToken)
		}
		return []byte(maker.config.AuthConfig.JwtSecretKey), nil
	}

	token, err := jwt.Parse(oldToken, keyFunc)
	if err != nil {
		maker.logger.ErrorWithID(ctx, "[Utils: JWT] Error parsing token", "error", err)
		return "", nil, app_error.New(constants.ErrInvalidToken, app_error.ErrCodeAuthInvalidToken)
	}

	// Extract payload from parsed token
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		maker.logger.ErrorWithID(ctx, "[Utils: JWT] Invalid token claims type", "error", constants.ErrInvalidToken)
		return "", nil, app_error.New(constants.ErrInvalidToken, app_error.ErrCodeAuthInvalidToken)
	}

	payload := &VerifyEmailTokenPayload{}
	if idStr, ok := claims["id"].(string); ok {
		if id, err := uuid.Parse(idStr); err == nil {
			payload.ID = id
		}
	}
	if userID, ok := claims["user_id"].(string); ok {
		payload.UserID = userID
	}
	if email, ok := claims["email"].(string); ok {
		payload.Email = email
	}
	if issuedAt, ok := claims["issued_at"].(float64); ok {
		payload.IssuedAt = time.Unix(int64(issuedAt), 0)
	}
	if expiredAt, ok := claims["expires_at"].(float64); ok {
		payload.ExpiredAt = time.Unix(int64(expiredAt), 0)
	}

	payload.ExpiredAt = time.Now().Add(maker.config.AuthConfig.VerifyEmailTokenDuration)

	newToken, err := maker.createJWTToken(ctx, payload)
	if err != nil {
		return "", nil, err
	}

	return newToken, payload, nil
}
