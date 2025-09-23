package services

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
	db "gitlab.com/interview-simulation/interview-backend-server/internal/db/sqlc"
	"gitlab.com/interview-simulation/interview-backend-server/internal/entities"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
	"gitlab.com/interview-simulation/interview-backend-server/internal/middleware"
	mockMiddleware "gitlab.com/interview-simulation/interview-backend-server/internal/mocks/middleware"
	mockRepositories "gitlab.com/interview-simulation/interview-backend-server/internal/mocks/repositories"
	mockGenerator "gitlab.com/interview-simulation/interview-backend-server/internal/mocks/utils"
	utilsPkg "gitlab.com/interview-simulation/interview-backend-server/internal/utils"
)

func TestReviewCommentService_CreateReviewComment(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()
	globalID := "01234567-89ab-cdef-0123-456789abcdef"
	mockErr := errors.New("auth failed")
	invalidID := "invalid-id"

	testCases := []struct {
		name   string
		input  *entities.CreateReviewCommentReq
		setup  func() (*mockMiddleware.MockAuthContext, *mockRepositories.MockReviewCommentRepository, *mockGenerator.MockGenerator)
		verify func(t *testing.T, gotErr error)
	}{
		{
			name: "Success",
			input: &entities.CreateReviewCommentReq{
				SessionID: globalID,
				Rating:    5,
				Comment:   "Great job!",
			},
			setup: func() (*mockMiddleware.MockAuthContext, *mockRepositories.MockReviewCommentRepository, *mockGenerator.MockGenerator) {
				mockReviewCommentRepository := new(mockRepositories.MockReviewCommentRepository)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockGenerator := new(mockGenerator.MockGenerator)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							UserID: globalID,
							Role:   constants.UserRoleUser,
						},
					}, nil)

				mockGenerator.EXPECT().
					GenerateUUID(ctx).
					Return(uuid.MustParse(globalID))

				mockReviewCommentRepository.EXPECT().
					CreateReviewComment(ctx, mock.MatchedBy(func(req *db.CreateReviewCommentParams) bool {
						return req.ID == uuid.MustParse(globalID) &&
							req.SessionID == uuid.MustParse(globalID) &&
							req.AuthorType == string(constants.UserRoleUser) &&
							req.AuthorUserID.UUID == uuid.MustParse(globalID) &&
							req.Rating.Int16 == 5 &&
							req.Description.String == "Great job!"
					})).
					Return(nil)

				return mockAuthContext, mockReviewCommentRepository, mockGenerator
			},
			verify: func(t *testing.T, gotErr error) {
				assert.NoError(t, gotErr)
			},
		},
		{
			name: "Error WithParseSessionIDError",
			input: &entities.CreateReviewCommentReq{
				SessionID: invalidID,
				Rating:    5,
				Comment:   "Great job!",
			},
			setup: func() (*mockMiddleware.MockAuthContext, *mockRepositories.MockReviewCommentRepository, *mockGenerator.MockGenerator) {
				mockReviewCommentRepository := new(mockRepositories.MockReviewCommentRepository)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockGenerator := new(mockGenerator.MockGenerator)

				return mockAuthContext, mockReviewCommentRepository, mockGenerator
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "The UUID is invalid. Please try again.")
				assert.Contains(t, gotErr.Error(), "[INS0107]")
			},
		},
		{
			name: "Error WithGetAuthContextError",
			input: &entities.CreateReviewCommentReq{
				SessionID: globalID,
				Rating:    5,
				Comment:   "Great job!",
			},
			setup: func() (*mockMiddleware.MockAuthContext, *mockRepositories.MockReviewCommentRepository, *mockGenerator.MockGenerator) {
				mockReviewCommentRepository := new(mockRepositories.MockReviewCommentRepository)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockGenerator := new(mockGenerator.MockGenerator)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(nil, mockErr)

				return mockAuthContext, mockReviewCommentRepository, mockGenerator
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.ErrorIs(t, gotErr, mockErr)
			},
		},
		{
			name: "Error WithParseUserIDError",
			input: &entities.CreateReviewCommentReq{
				SessionID: globalID,
				Rating:    5,
				Comment:   "Great job!",
			},
			setup: func() (*mockMiddleware.MockAuthContext, *mockRepositories.MockReviewCommentRepository, *mockGenerator.MockGenerator) {
				mockReviewCommentRepository := new(mockRepositories.MockReviewCommentRepository)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockGenerator := new(mockGenerator.MockGenerator)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							UserID: invalidID,
							Role:   constants.UserRoleUser,
						},
					}, nil)

				return mockAuthContext, mockReviewCommentRepository, mockGenerator
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "The UUID is invalid. Please try again.")
				assert.Contains(t, gotErr.Error(), "[INS0107]")
			},
		},
		{
			name: "Error WithCreateReviewCommentError",
			input: &entities.CreateReviewCommentReq{
				SessionID: globalID,
				Rating:    5,
				Comment:   "Great job!",
			},
			setup: func() (*mockMiddleware.MockAuthContext, *mockRepositories.MockReviewCommentRepository, *mockGenerator.MockGenerator) {
				mockReviewCommentRepository := new(mockRepositories.MockReviewCommentRepository)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockGenerator := new(mockGenerator.MockGenerator)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							UserID: globalID,
							Role:   constants.UserRoleUser,
						},
					}, nil)

				mockGenerator.EXPECT().
					GenerateUUID(ctx).
					Return(uuid.MustParse(globalID))

				mockReviewCommentRepository.EXPECT().
					CreateReviewComment(ctx, mock.MatchedBy(func(req *db.CreateReviewCommentParams) bool {
						return req.ID == uuid.MustParse(globalID) &&
							req.SessionID == uuid.MustParse(globalID) &&
							req.AuthorType == string(constants.UserRoleUser) &&
							req.AuthorUserID.UUID == uuid.MustParse(globalID) &&
							req.Rating.Int16 == 5 &&
							req.Description.String == "Great job!"
					})).
					Return(mockErr)

				return mockAuthContext, mockReviewCommentRepository, mockGenerator
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.ErrorIs(t, gotErr, mockErr)
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			mockAuthContext, mockReviewCommentRepository, mockGenerator := tC.setup()
			defer func() {
				if mockAuthContext != nil {
					mockAuthContext.AssertExpectations(t)
				}
				if mockReviewCommentRepository != nil {
					mockReviewCommentRepository.AssertExpectations(t)
				}
				if mockGenerator != nil {
					mockGenerator.AssertExpectations(t)
				}
			}()

			svc := NewReviewCommentService(lgr, mockReviewCommentRepository, mockAuthContext, mockGenerator)
			gotErr := svc.CreateReviewComment(ctx, tC.input)

			tC.verify(t, gotErr)
		})
	}
}
