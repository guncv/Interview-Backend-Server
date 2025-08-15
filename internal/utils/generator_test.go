package utils

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
)

func TestGenerateUUID(t *testing.T) {
	logger := log.Initialize(constants.TestAppEnv)

	svc := NewGenerator(logger)
	uuid := svc.GenerateUUID(context.Background())
	assert.NotEmpty(t, uuid)
}
