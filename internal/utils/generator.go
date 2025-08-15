package utils

import (
	"context"

	"github.com/google/uuid"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
)

type Generator interface {
	GenerateUUID(ctx context.Context) uuid.UUID
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
	g.log.InfoWithID(ctx, "[Utils: GenerateUUID] Called")
	uuid := uuid.New()

	return uuid
}
