package utils

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	mathrand "math/rand"

	"github.com/google/uuid"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
)

type Generator interface {
	GenerateUUID(ctx context.Context) uuid.UUID
	GenerateRandomString(ctx context.Context, length int) string
	GenerateCryptographicallySecureString(ctx context.Context, length int) string
}

type generator struct {
	log *log.Logger
}

func NewGenerator(log *log.Logger) Generator {
	return &generator{
		log: log,
	}
}

func (g *generator) GenerateUUID(ctx context.Context) uuid.UUID {
	uuid := uuid.New()

	return uuid
}

func (g *generator) GenerateRandomString(ctx context.Context, length int) string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)

	for i := range b {
		b[i] = charset[mathrand.Intn(len(charset))]
	}

	return string(b)
}

func (g *generator) GenerateCryptographicallySecureString(ctx context.Context, length int) string {
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)

	if err != nil {
		g.log.Error(ctx, "Failed to generate cryptographically secure random bytes, falling back to math/rand", "error", err)
		return g.GenerateRandomString(ctx, length)
	}

	return hex.EncodeToString(bytes)
}
