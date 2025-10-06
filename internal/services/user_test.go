package services

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
	"gitlab.com/interview-simulation/interview-backend-server/internal/entities"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
	"gitlab.com/interview-simulation/interview-backend-server/internal/middleware"
	middlewareMocks "gitlab.com/interview-simulation/interview-backend-server/internal/mocks/middleware"
	"gitlab.com/interview-simulation/interview-backend-server/internal/mocks/repositories"
	utilsPkg "gitlab.com/interview-simulation/interview-backend-server/internal/utils"
)

func TestUserService_HealthCheck(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()
	mockErr := errors.New("error")

	okResponse := entities.HealthCheckResponse{
		Status: "ok",
	}
	testCases := []struct {
		name   string
		setup  func() *repositories.MockUserRepository
		verify func(t *testing.T, got entities.HealthCheckResponse, gotErr error)
	}{
		{
			name: "OK",
			setup: func() *repositories.MockUserRepository {
				mockUserRepo := new(repositories.MockUserRepository)
				mockUserRepo.EXPECT().
					HealthCheck(ctx).
					Return("ok", nil)

				return mockUserRepo
			},
			verify: func(t *testing.T, got entities.HealthCheckResponse, gotErr error) {
				assert.Equal(t, okResponse, got)
				assert.NoError(t, gotErr)
			},
		},
		{
			name: "Error",
			setup: func() *repositories.MockUserRepository {
				mockUserRepo := new(repositories.MockUserRepository)
				mockUserRepo.EXPECT().
					HealthCheck(ctx).
					Return("", mockErr)

				return mockUserRepo
			},
			verify: func(t *testing.T, got entities.HealthCheckResponse, gotErr error) {
				assert.Equal(t, entities.HealthCheckResponse{}, got)
				assert.Error(t, gotErr)
				assert.ErrorIs(t, gotErr, mockErr)
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			mockUserRepo := tC.setup()
			defer mockUserRepo.AssertExpectations(t)

			svc := NewUserService(lgr, mockUserRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil)
			got, gotErr := svc.HealthCheck(ctx)

			tC.verify(t, got, gotErr)
		})
	}
}

func TestUserService_SignOut(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()
	mockErr := errors.New("error")

	testCases := []struct {
		name   string
		setup  func() (*repositories.MockAuthSessionRepository, *middlewareMocks.MockAuthContext)
		verify func(t *testing.T, gotErr error)
	}{
		{
			name: "Success",
			setup: func() (*repositories.MockAuthSessionRepository, *middlewareMocks.MockAuthContext) {
				mockAuthSessionRepo := new(repositories.MockAuthSessionRepository)
				mockAuthContext := new(middlewareMocks.MockAuthContext)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
						},
					}, nil)

				// Mock session revocation
				mockAuthSessionRepo.EXPECT().
					RevokeAuthSessionByID(ctx, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")).
					Return(nil)

				return mockAuthSessionRepo, mockAuthContext
			},
			verify: func(t *testing.T, gotErr error) {
				assert.NoError(t, gotErr)
			},
		},
		{
			name: "Error_AuthContextFailed",
			setup: func() (*repositories.MockAuthSessionRepository, *middlewareMocks.MockAuthContext) {
				mockAuthContext := new(middlewareMocks.MockAuthContext)

				// Mock auth context retrieval fails
				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(nil, mockErr)

				return nil, mockAuthContext
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
			},
		},
		{
			name: "Error_SessionRevocationFailed",
			setup: func() (*repositories.MockAuthSessionRepository, *middlewareMocks.MockAuthContext) {
				mockAuthSessionRepo := new(repositories.MockAuthSessionRepository)
				mockAuthContext := new(middlewareMocks.MockAuthContext)

				// Mock auth context retrieval
				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
						},
					}, nil)

				// Mock session revocation fails
				mockAuthSessionRepo.EXPECT().
					RevokeAuthSessionByID(ctx, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")).
					Return(mockErr)

				return mockAuthSessionRepo, mockAuthContext
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.ErrorIs(t, gotErr, mockErr)
			},
		},
		{
			name: "Error_InvalidAuthPayload",
			setup: func() (*repositories.MockAuthSessionRepository, *middlewareMocks.MockAuthContext) {
				mockAuthContext := new(middlewareMocks.MockAuthContext)

				// Mock auth context retrieval returns invalid payload
				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: nil,
					}, nil)

				return nil, mockAuthContext
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
			},
		},
		{
			name: "Error_NilAuthContext",
			setup: func() (*repositories.MockAuthSessionRepository, *middlewareMocks.MockAuthContext) {
				mockAuthContext := new(middlewareMocks.MockAuthContext)

				// Mock auth context retrieval returns nil
				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(nil, nil)

				return nil, mockAuthContext
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
			},
		},
		{
			name: "Error_InvalidSessionID",
			setup: func() (*repositories.MockAuthSessionRepository, *middlewareMocks.MockAuthContext) {
				mockAuthSessionRepo := new(repositories.MockAuthSessionRepository)
				mockAuthContext := new(middlewareMocks.MockAuthContext)

				// Mock auth context retrieval returns invalid session ID
				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							ID: uuid.Nil,
						},
					}, nil)

				return mockAuthSessionRepo, mockAuthContext
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			mockAuthSessionRepo, mockAuthContext := tC.setup()
			defer func() {
				if mockAuthSessionRepo != nil {
					mockAuthSessionRepo.AssertExpectations(t)
				}
				if mockAuthContext != nil {
					mockAuthContext.AssertExpectations(t)
				}
			}()

			svc := NewUserService(lgr, nil, mockAuthSessionRepo, nil, nil, nil, mockAuthContext, nil, nil, nil, nil)
			gotErr := svc.SignOut(ctx)

			tC.verify(t, gotErr)
		})
	}
}
