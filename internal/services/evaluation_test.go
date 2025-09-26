package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
	db "gitlab.com/interview-simulation/interview-backend-server/internal/db/sqlc"
	"gitlab.com/interview-simulation/interview-backend-server/internal/entities"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/database"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
	mockDatabase "gitlab.com/interview-simulation/interview-backend-server/internal/mocks/database"
	mockRepositories "gitlab.com/interview-simulation/interview-backend-server/internal/mocks/repositories"
	mockUtils "gitlab.com/interview-simulation/interview-backend-server/internal/mocks/utils"
	"gitlab.com/interview-simulation/interview-backend-server/internal/repositories"
)

func TestEvaluationService_GetRubricWithCriteriaByName(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()

	globalID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	validResp := &entities.GetRubricWithCriteriaByNameResp{
		RubricID:            globalID.String(),
		RubricName:          "test",
		RubricDescriptionMd: "test",
		RubricVersionLabel:  "test",
		Criteria: []entities.CritetiaRow{
			{
				CriterionID:            globalID.String(),
				CriterionCode:          "test",
				CriterionName:          "test",
				CriterionDescriptionMd: "test",
				CriterionWeight:        "test",
				CriterionMaxScore:      "test",
			},
		},
	}

	jsonValidResp, err := json.Marshal(validResp)
	if err != nil {
		t.Fatalf("Failed to marshal valid response: %v", err)
	}

	testCases := []struct {
		name   string
		input  string
		setup  func() (*mockDatabase.MockRedisClient, *mockRepositories.MockEvaluationRubricsRepository)
		verify func(t *testing.T, gotResp *entities.GetRubricWithCriteriaByNameResp, gotErr error)
	}{
		{
			name:  "Success - WithCacheHit",
			input: "test",
			setup: func() (*mockDatabase.MockRedisClient, *mockRepositories.MockEvaluationRubricsRepository) {
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s:%s", constants.RedisPrefixEvaluationRubric, "test")).
					Return(string(jsonValidResp), nil)

				return mockRedisClient, nil
			},
			verify: func(t *testing.T, gotResp *entities.GetRubricWithCriteriaByNameResp, gotErr error) {
				assert.NoError(t, gotErr)
				assert.NotNil(t, gotResp)
				assert.Equal(t, validResp, gotResp)
			},
		},
		{
			name:  "Success - WithCacheMiss",
			input: "test",
			setup: func() (*mockDatabase.MockRedisClient, *mockRepositories.MockEvaluationRubricsRepository) {
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockEvaluationRubricsRepo := new(mockRepositories.MockEvaluationRubricsRepository)

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s:%s", constants.RedisPrefixEvaluationRubric, "test")).
					Return("", errors.New("cache miss"))

				repoReq := &db.GetRubricWithCriteriaByNameParams{
					Name:         constants.StateToCriteriaStateMap["test"],
					VersionLabel: constants.CurrentCriteriaVersion,
				}

				repoResp := []db.GetRubricWithCriteriaByNameRow{
					{
						RubricID:               globalID,
						RubricName:             "test",
						RubricDescriptionMd:    sql.NullString{String: "test", Valid: true},
						RubricVersionLabel:     "test",
						CriterionID:            globalID,
						CriterionCode:          "test",
						CriterionName:          "test",
						CriterionDescriptionMd: sql.NullString{String: "test", Valid: true},
						CriterionWeight:        "test",
						CriterionMaxScore:      "test",
					},
				}

				mockEvaluationRubricsRepo.EXPECT().
					GetRubricWithCriteriaByName(ctx, repoReq).
					Return(repoResp, nil)

				mockRedisClient.EXPECT().
					Set(mock.AnythingOfType("*context.timerCtx"), database.RedisPayload{
						Key:   fmt.Sprintf("%s:%s", constants.RedisPrefixEvaluationRubric, "test"),
						Value: jsonValidResp,
						TTL:   constants.RedisTTLEvaluationRubric,
					}).
					Return(nil).
					Maybe()

				return mockRedisClient, mockEvaluationRubricsRepo
			},
			verify: func(t *testing.T, gotResp *entities.GetRubricWithCriteriaByNameResp, gotErr error) {
				assert.NoError(t, gotErr)
				assert.NotNil(t, gotResp)
				assert.Equal(t, validResp, gotResp)
			},
		},
		{
			name:  "Error - WithCacheMissAndDBError",
			input: "test",
			setup: func() (*mockDatabase.MockRedisClient, *mockRepositories.MockEvaluationRubricsRepository) {
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockEvaluationRubricsRepo := new(mockRepositories.MockEvaluationRubricsRepository)

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s:%s", constants.RedisPrefixEvaluationRubric, "test")).
					Return("", errors.New("cache miss"))

				repoReq := &db.GetRubricWithCriteriaByNameParams{
					Name:         constants.StateToCriteriaStateMap["test"],
					VersionLabel: constants.CurrentCriteriaVersion,
				}

				mockEvaluationRubricsRepo.EXPECT().
					GetRubricWithCriteriaByName(ctx, repoReq).
					Return(nil, errors.New("db error"))

				return mockRedisClient, mockEvaluationRubricsRepo
			},
			verify: func(t *testing.T, gotResp *entities.GetRubricWithCriteriaByNameResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, gotResp)
			},
		},
		{
			name:  "Error - WithCacheMissAndNoRubricFoundInDB",
			input: "test",
			setup: func() (*mockDatabase.MockRedisClient, *mockRepositories.MockEvaluationRubricsRepository) {
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockEvaluationRubricsRepo := new(mockRepositories.MockEvaluationRubricsRepository)

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s:%s", constants.RedisPrefixEvaluationRubric, "test")).
					Return("", errors.New("cache miss"))

				repoReq := &db.GetRubricWithCriteriaByNameParams{
					Name:         constants.StateToCriteriaStateMap["test"],
					VersionLabel: constants.CurrentCriteriaVersion,
				}

				mockEvaluationRubricsRepo.EXPECT().
					GetRubricWithCriteriaByName(ctx, repoReq).
					Return([]db.GetRubricWithCriteriaByNameRow{}, nil)

				return mockRedisClient, mockEvaluationRubricsRepo
			},
			verify: func(t *testing.T, gotResp *entities.GetRubricWithCriteriaByNameResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, gotResp)
				assert.Equal(t, app_error.ErrCodeEvaluationRubricCriteriaNotFound, gotErr.(*app_error.AppError).Code)
			},
		},
		{
			name:  "Success - WithCacheHitAndUnmarshalFailed",
			input: "test",
			setup: func() (*mockDatabase.MockRedisClient, *mockRepositories.MockEvaluationRubricsRepository) {
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s:%s", constants.RedisPrefixEvaluationRubric, "test")).
					Return("invalid json", nil)

				return mockRedisClient, nil
			},
			verify: func(t *testing.T, gotResp *entities.GetRubricWithCriteriaByNameResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, gotResp)
				assert.Equal(t, app_error.ErrCodeGeneralUnmarshalFailed, gotErr.(*app_error.AppError).Code)
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			mockRedisClient, mockEvaluationRubricsRepo := tC.setup()

			defer func() {
				if mockRedisClient != nil {
					mockRedisClient.AssertExpectations(t)
				}
				if mockEvaluationRubricsRepo != nil {
					mockEvaluationRubricsRepo.AssertExpectations(t)
				}
			}()

			svc := NewEvaluationService(
				lgr,
				mockRedisClient,
				mockEvaluationRubricsRepo,
				nil,
				nil,
				nil,
				nil,
				nil,
			)

			gotResp, gotErr := svc.GetRubricWithCriteriaByName(ctx, tC.input)

			tC.verify(t, gotResp, gotErr)
		})
	}
}

func TestEvaluationService_ListAllRubricsAndCriteria(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()
	globalID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	validLowWeightDBResp := []db.ListAllRubricsAndCriteriaRow{
		{
			RubricID:               globalID,
			RubricName:             "test",
			RubricDescriptionMd:    sql.NullString{String: "test", Valid: true},
			CriterionID:            globalID,
			CriterionName:          "test",
			CriterionDescriptionMd: sql.NullString{String: "test", Valid: true},
			CriterionWeight:        "0.20",
		},
	}

	validMediumWeightDBResp := validLowWeightDBResp
	validMediumWeightDBResp[0].CriterionWeight = "0.30"

	validHighWeightDBResp := validLowWeightDBResp
	validHighWeightDBResp[0].CriterionWeight = "0.40"

	validLowWeightResp := &entities.ListAllRubricsAndCriteriaResp{
		Rubrics: []entities.RubricAndCriteriaRow{
			{
				ID:            globalID.String(),
				Name:          "test",
				DescriptionMd: "test",
				Criteria: []entities.CriterionRow{
					{
						ID:            globalID.String(),
						Name:          "test",
						DescriptionMd: "test",
						Percentage:    "20.0",
						Color:         constants.PercentageColorLow,
					},
				},
			},
		},
	}

	validMediumWeightResp := validLowWeightResp
	validMediumWeightResp.Rubrics[0].Criteria[0].Percentage = "30.0"
	validMediumWeightResp.Rubrics[0].Criteria[0].Color = constants.PercentageColorMedium

	validHighWeightResp := validLowWeightResp
	validHighWeightResp.Rubrics[0].Criteria[0].Percentage = "40.0"
	validHighWeightResp.Rubrics[0].Criteria[0].Color = constants.PercentageColorHigh

	jsonLowWeightValidResp, err := json.Marshal(validLowWeightResp)
	if err != nil {
		t.Fatalf("Failed to marshal valid response: %v", err)
	}

	jsonMediumWeightValidResp, err := json.Marshal(validMediumWeightResp)
	if err != nil {
		t.Fatalf("Failed to marshal valid response: %v", err)
	}

	jsonHighWeightValidResp, err := json.Marshal(validHighWeightResp)
	if err != nil {
		t.Fatalf("Failed to marshal valid response: %v", err)
	}

	testCases := []struct {
		name   string
		setup  func() (*mockDatabase.MockRedisClient, *mockRepositories.MockEvaluationRubricsRepository)
		verify func(t *testing.T, gotResp *entities.ListAllRubricsAndCriteriaResp, gotErr error)
	}{
		{
			name: "Success WithCacheHit",
			setup: func() (*mockDatabase.MockRedisClient, *mockRepositories.MockEvaluationRubricsRepository) {
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockRedisClient.EXPECT().
					Get(ctx, constants.RedisPrefixAllRubricsAndCriteria).
					Return(string(jsonLowWeightValidResp), nil)

				return mockRedisClient, nil
			},
			verify: func(t *testing.T, gotResp *entities.ListAllRubricsAndCriteriaResp, gotErr error) {
				assert.NoError(t, gotErr)
				assert.NotNil(t, gotResp)
				assert.Equal(t, validLowWeightResp, gotResp)
			},
		},
		{
			name: "Success WithCacheError",
			setup: func() (*mockDatabase.MockRedisClient, *mockRepositories.MockEvaluationRubricsRepository) {
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockEvaluationRubricsRepo := new(mockRepositories.MockEvaluationRubricsRepository)

				mockRedisClient.EXPECT().
					Get(ctx, constants.RedisPrefixAllRubricsAndCriteria).
					Return("", errors.New("cache error"))

				mockEvaluationRubricsRepo.EXPECT().
					ListAllRubricsAndCriteria(ctx, constants.CurrentCriteriaVersion).
					Return(validLowWeightDBResp, nil)

				mockRedisClient.EXPECT().
					Set(mock.AnythingOfType("*context.timerCtx"), database.RedisPayload{
						Key:   constants.RedisPrefixAllRubricsAndCriteria,
						Value: jsonLowWeightValidResp,
						TTL:   constants.RedisTTLEvaluationRubric,
					}).
					Return(nil).
					Maybe()

				return mockRedisClient, mockEvaluationRubricsRepo
			},
			verify: func(t *testing.T, gotResp *entities.ListAllRubricsAndCriteriaResp, gotErr error) {
				assert.NoError(t, gotErr)
				assert.NotNil(t, gotResp)
				assert.Equal(t, validLowWeightResp, gotResp)
			},
		},
		{
			name: "Success WithCacheMissAndUnmarshalFailed",
			setup: func() (*mockDatabase.MockRedisClient, *mockRepositories.MockEvaluationRubricsRepository) {
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockEvaluationRubricsRepo := new(mockRepositories.MockEvaluationRubricsRepository)

				mockRedisClient.EXPECT().
					Get(ctx, constants.RedisPrefixAllRubricsAndCriteria).
					Return("invalid json", nil)

				mockEvaluationRubricsRepo.EXPECT().
					ListAllRubricsAndCriteria(ctx, constants.CurrentCriteriaVersion).
					Return(validLowWeightDBResp, nil)

				mockRedisClient.EXPECT().
					Set(mock.AnythingOfType("*context.timerCtx"), database.RedisPayload{
						Key:   constants.RedisPrefixAllRubricsAndCriteria,
						Value: jsonLowWeightValidResp,
						TTL:   constants.RedisTTLEvaluationRubric,
					}).
					Return(nil).
					Maybe()

				return mockRedisClient, mockEvaluationRubricsRepo
			},
			verify: func(t *testing.T, gotResp *entities.ListAllRubricsAndCriteriaResp, gotErr error) {
				assert.NoError(t, gotErr)
				assert.NotNil(t, gotResp)
				assert.Equal(t, validLowWeightResp, gotResp)
			},
		},
		{
			name: "Success WithCacheMiss",
			setup: func() (*mockDatabase.MockRedisClient, *mockRepositories.MockEvaluationRubricsRepository) {
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockEvaluationRubricsRepo := new(mockRepositories.MockEvaluationRubricsRepository)

				mockRedisClient.EXPECT().
					Get(ctx, constants.RedisPrefixAllRubricsAndCriteria).
					Return("", redis.Nil)

				mockEvaluationRubricsRepo.EXPECT().
					ListAllRubricsAndCriteria(ctx, constants.CurrentCriteriaVersion).
					Return(validLowWeightDBResp, nil)

				mockRedisClient.EXPECT().
					Set(mock.AnythingOfType("*context.timerCtx"), database.RedisPayload{
						Key:   constants.RedisPrefixAllRubricsAndCriteria,
						Value: jsonLowWeightValidResp,
						TTL:   constants.RedisTTLEvaluationRubric,
					}).
					Return(nil).
					Maybe()

				return mockRedisClient, mockEvaluationRubricsRepo
			},
			verify: func(t *testing.T, gotResp *entities.ListAllRubricsAndCriteriaResp, gotErr error) {
				assert.NoError(t, gotErr)
				assert.NotNil(t, gotResp)
				assert.Equal(t, validLowWeightResp, gotResp)
			},
		},
		{
			name: "Success WithCacheMissAndMediumWeight",
			setup: func() (*mockDatabase.MockRedisClient, *mockRepositories.MockEvaluationRubricsRepository) {
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockEvaluationRubricsRepo := new(mockRepositories.MockEvaluationRubricsRepository)

				mockRedisClient.EXPECT().
					Get(ctx, constants.RedisPrefixAllRubricsAndCriteria).
					Return("", redis.Nil)

				mockEvaluationRubricsRepo.EXPECT().
					ListAllRubricsAndCriteria(ctx, constants.CurrentCriteriaVersion).
					Return(validMediumWeightDBResp, nil)

				mockRedisClient.EXPECT().
					Set(mock.AnythingOfType("*context.timerCtx"), database.RedisPayload{
						Key:   constants.RedisPrefixAllRubricsAndCriteria,
						Value: jsonMediumWeightValidResp,
						TTL:   constants.RedisTTLEvaluationRubric,
					}).
					Return(nil).
					Maybe()

				return mockRedisClient, mockEvaluationRubricsRepo
			},
			verify: func(t *testing.T, gotResp *entities.ListAllRubricsAndCriteriaResp, gotErr error) {
				assert.NoError(t, gotErr)
				assert.NotNil(t, gotResp)
				assert.Equal(t, validMediumWeightResp, gotResp)
			},
		},
		{
			name: "Success WithCacheMissAndHighWeight",
			setup: func() (*mockDatabase.MockRedisClient, *mockRepositories.MockEvaluationRubricsRepository) {
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockEvaluationRubricsRepo := new(mockRepositories.MockEvaluationRubricsRepository)

				mockRedisClient.EXPECT().
					Get(ctx, constants.RedisPrefixAllRubricsAndCriteria).
					Return("", redis.Nil)

				mockEvaluationRubricsRepo.EXPECT().
					ListAllRubricsAndCriteria(ctx, constants.CurrentCriteriaVersion).
					Return(validHighWeightDBResp, nil)

				mockRedisClient.EXPECT().
					Set(mock.AnythingOfType("*context.timerCtx"), database.RedisPayload{
						Key:   constants.RedisPrefixAllRubricsAndCriteria,
						Value: jsonHighWeightValidResp,
						TTL:   constants.RedisTTLEvaluationRubric,
					}).
					Return(nil).
					Maybe()

				return mockRedisClient, mockEvaluationRubricsRepo
			},
			verify: func(t *testing.T, gotResp *entities.ListAllRubricsAndCriteriaResp, gotErr error) {
				assert.NoError(t, gotErr)
				assert.NotNil(t, gotResp)
				assert.Equal(t, validHighWeightResp, gotResp)
			},
		},
		{
			name: "Success WithCacheMissAndGetEmptyResponseFromDB",
			setup: func() (*mockDatabase.MockRedisClient, *mockRepositories.MockEvaluationRubricsRepository) {
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockEvaluationRubricsRepo := new(mockRepositories.MockEvaluationRubricsRepository)

				mockRedisClient.EXPECT().
					Get(ctx, constants.RedisPrefixAllRubricsAndCriteria).
					Return("", redis.Nil)

				mockEvaluationRubricsRepo.EXPECT().
					ListAllRubricsAndCriteria(ctx, constants.CurrentCriteriaVersion).
					Return([]db.ListAllRubricsAndCriteriaRow{}, nil)

				return mockRedisClient, mockEvaluationRubricsRepo
			},
			verify: func(t *testing.T, gotResp *entities.ListAllRubricsAndCriteriaResp, gotErr error) {
				assert.NoError(t, gotErr)
				assert.NotNil(t, gotResp)
				assert.Equal(t, &entities.ListAllRubricsAndCriteriaResp{
					Rubrics: []entities.RubricAndCriteriaRow{},
				}, gotResp)
			},
		},
		{
			name: "Error WithCacheMissAndDBError",
			setup: func() (*mockDatabase.MockRedisClient, *mockRepositories.MockEvaluationRubricsRepository) {
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockEvaluationRubricsRepo := new(mockRepositories.MockEvaluationRubricsRepository)

				mockRedisClient.EXPECT().
					Get(ctx, constants.RedisPrefixAllRubricsAndCriteria).
					Return("", errors.New("cache miss"))

				mockEvaluationRubricsRepo.EXPECT().
					ListAllRubricsAndCriteria(ctx, constants.CurrentCriteriaVersion).
					Return(nil, errors.New("db error"))

				return mockRedisClient, mockEvaluationRubricsRepo
			},
			verify: func(t *testing.T, gotResp *entities.ListAllRubricsAndCriteriaResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Equal(t, gotErr.Error(), "db error")
				assert.Nil(t, gotResp)

			},
		},
		{
			name: "Error WithCacheHitButUnmarshalFailedAndDBError",
			setup: func() (*mockDatabase.MockRedisClient, *mockRepositories.MockEvaluationRubricsRepository) {
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockEvaluationRubricsRepo := new(mockRepositories.MockEvaluationRubricsRepository)

				mockRedisClient.EXPECT().
					Get(ctx, constants.RedisPrefixAllRubricsAndCriteria).
					Return("invalid json", nil)

				mockEvaluationRubricsRepo.EXPECT().
					ListAllRubricsAndCriteria(ctx, constants.CurrentCriteriaVersion).
					Return(nil, errors.New("db error"))

				return mockRedisClient, mockEvaluationRubricsRepo
			},
			verify: func(t *testing.T, gotResp *entities.ListAllRubricsAndCriteriaResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Equal(t, gotErr.Error(), "db error")
				assert.Nil(t, gotResp)
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			mockRedisClient, mockEvaluationRubricsRepo := tC.setup()

			defer func() {
				if mockRedisClient != nil {
					mockRedisClient.AssertExpectations(t)
				}
				if mockEvaluationRubricsRepo != nil {
					mockEvaluationRubricsRepo.AssertExpectations(t)
				}
			}()

			svc := NewEvaluationService(
				lgr,
				mockRedisClient,
				mockEvaluationRubricsRepo,
				nil,
				nil,
				nil,
				nil,
				nil,
			)

			gotResp, gotErr := svc.ListAllRubricsAndCriteria(ctx)

			tC.verify(t, gotResp, gotErr)
		})
	}
}

func TestEvaluationService_CalculateTurnScore(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()

	// Test data
	validSessionID := "550e8400-e29b-41d4-a716-446655440000"
	validUserTurnID := "550e8400-e29b-41d4-a716-446655440001"
	validUserID := "550e8400-e29b-41d4-a716-446655440002"
	validRubricID := "550e8400-e29b-41d4-a716-446655440003"
	validCriterionID := "550e8400-e29b-41d4-a716-446655440004"
	validEvaluationID := "550e8400-e29b-41d4-a716-446655440005"
	validImproveSentenceID := "550e8400-e29b-41d4-a716-446655440006"

	validReq := &entities.CalculateTurnScoreReq{
		SessionID:          validSessionID,
		UserTurnID:         validUserTurnID,
		UserID:             validUserID,
		UserMessage:        "Test user message",
		InterviewerMessage: "Test interviewer message",
		CurrentState:       "technical",
		CurrentStateID:     "550e8400-e29b-41d4-a716-446655440007",
	}

	validRubric := &entities.GetRubricWithCriteriaByNameResp{
		RubricID:            validRubricID,
		RubricName:          "Technical Skills",
		RubricDescriptionMd: "Technical skills evaluation",
		RubricVersionLabel:  "v1.0",
		Criteria: []entities.CritetiaRow{
			{
				CriterionID:            validCriterionID,
				CriterionCode:          "TECH_001",
				CriterionName:          "Problem Solving",
				CriterionDescriptionMd: "Problem solving ability",
				CriterionWeight:        "0.5",
				CriterionMaxScore:      "10",
			},
		},
	}

	invalidRubric := &entities.GetRubricWithCriteriaByNameResp{
		RubricID:            "invalid-rubric-id",
		RubricName:          "Technical Skills",
		RubricDescriptionMd: "Technical skills evaluation",
		RubricVersionLabel:  "v1.0",
		Criteria: []entities.CritetiaRow{
			{
				CriterionID:            validCriterionID,
				CriterionCode:          "TECH_001",
				CriterionName:          "Problem Solving",
				CriterionDescriptionMd: "Problem solving ability",
				CriterionWeight:        "0.5",
				CriterionMaxScore:      "10",
			},
		},
	}

	invalidCriterionResp := &repositories.InterviewFeedbackAndScoreResp{
		OverallScore:    8.5,
		OverallFeedback: "Good technical skills demonstrated",
		ImproveSentence: "Try to explain your approach more clearly",
		LLmModel:        "gpt-4",
		CriteriaScores: []repositories.CriteriaScore{
			{
				CriterionID:       "invalid-criterion-id",
				CriterionName:     "Problem Solving",
				CriterionScore:    8,
				CriterionFeedback: "Good problem solving approach",
			},
		},
	}

	validInterviewFeedbackResp := &repositories.InterviewFeedbackAndScoreResp{
		OverallScore:    8.5,
		OverallFeedback: "Good technical skills demonstrated",
		ImproveSentence: "Try to explain your approach more clearly",
		LLmModel:        "gpt-4",
		CriteriaScores: []repositories.CriteriaScore{
			{
				CriterionID:       validCriterionID,
				CriterionName:     "Problem Solving",
				CriterionScore:    8,
				CriterionFeedback: "Good problem solving approach",
			},
		},
	}

	invalidInterviewFeedbackResp := &repositories.InterviewFeedbackAndScoreResp{
		OverallScore:    8.5,
		OverallFeedback: "Good technical skills demonstrated",
		ImproveSentence: "Try to explain your approach more clearly",
		LLmModel:        "gpt-4",
		CriteriaScores: []repositories.CriteriaScore{
			{
				CriterionID:       "invalid-criterion-id",
				CriterionName:     "Problem Solving",
				CriterionScore:    8,
				CriterionFeedback: "Good problem solving approach",
			},
		},
	}

	jsonInvalidRubric, err := json.Marshal(invalidRubric)
	if err != nil {
		t.Fatalf("Failed to marshal invalid rubric: %v", err)
	}

	jsonValidRubric, err := json.Marshal(validRubric)
	if err != nil {
		t.Fatalf("Failed to marshal valid rubric: %v", err)
	}

	testCases := []struct {
		name   string
		input  *entities.CalculateTurnScoreReq
		setup  func() (*mockDatabase.MockRedisClient, *mockRepositories.MockEvaluationRubricsRepository, *mockRepositories.MockEvaluationScoresRepository, *mockRepositories.MockInterviewTurnsRepository, *mockUtils.MockGenerator)
		verify func(t *testing.T, gotErr error)
	}{
		{
			name:  "Success - Complete Flow",
			input: validReq,
			setup: func() (*mockDatabase.MockRedisClient, *mockRepositories.MockEvaluationRubricsRepository, *mockRepositories.MockEvaluationScoresRepository, *mockRepositories.MockInterviewTurnsRepository, *mockUtils.MockGenerator) {
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockEvaluationRubricsRepo := new(mockRepositories.MockEvaluationRubricsRepository)
				mockEvaluationScoresRepo := new(mockRepositories.MockEvaluationScoresRepository)
				mockInterviewTurnsRepo := new(mockRepositories.MockInterviewTurnsRepository)
				mockGenerator := new(mockUtils.MockGenerator)

				// Mock GetRubricWithCriteriaByName (called internally) - cache hit
				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s:%s", constants.RedisPrefixEvaluationRubric, validReq.CurrentState)).
					Return(string(jsonValidRubric), nil)

				// Mock InterviewFeedbackAndScore
				mockEvaluationScoresRepo.EXPECT().
					InterviewFeedbackAndScore(ctx, mock.AnythingOfType("*repositories.InterviewFeedbackAndScoreReq")).
					Return(validInterviewFeedbackResp, nil)

				// Mock generator calls - need to match the order they're called
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(uuid.MustParse(validEvaluationID)).Times(1)
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(uuid.MustParse(validCriterionID)).Times(1)
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(uuid.MustParse(validImproveSentenceID)).Times(1)

				// Mock CreateEvaluationWithCriteriaScoreAndImproveSentenceTx
				mockEvaluationScoresRepo.EXPECT().
					CreateEvaluationWithCriteriaScoreAndImproveSentenceTx(ctx, mock.AnythingOfType("*repositories.CreateEvaluationAndScoreTxReq")).
					Return(nil)

				// Mock FlagIsScoreEvaluated
				mockInterviewTurnsRepo.EXPECT().
					FlagIsScoreEvaluated(ctx, uuid.MustParse(validUserTurnID)).
					Return(nil)

				// Mock Redis Get for last turn ID (with retries)
				redisKey := fmt.Sprintf("%s%s:%s", constants.RedisPrefixInterviewLastTurnID, validReq.SessionID, validReq.CurrentState)
				mockRedisClient.EXPECT().
					Get(ctx, redisKey).
					Return(validUserTurnID, nil)

				// Mock CalculateEvaluationInOldState (called when last turn ID matches)
				// This is called internally, so we need to mock the dependencies
				mockEvaluationScoresRepo.EXPECT().
					GetEvaluationSummaryJsonBySessionAndState(ctx, mock.AnythingOfType("*db.GetEvaluationSummaryJsonBySessionAndStateParams")).
					Return([]byte(`{"criteria":[],"evaluation_avg_score":8.5}`), nil)

				mockEvaluationScoresRepo.EXPECT().
					CalculateEachCriteriaCommentBySessionAndState(ctx, mock.AnythingOfType("*repositories.PreProcessedCriteriaReq")).
					Return(&repositories.PostProcessedCriteriaResp{
						Criteria: []repositories.PostProcessedCriteria{
							{
								CriteriaID:       validCriterionID,
								CriteriaName:     "Problem Solving",
								CriteriaAvgScore: 8.0,
								CriteriaComment:  "Good problem solving skills",
							},
						},
					}, nil)

				// Mock generator calls for CalculateEvaluationInOldState
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(uuid.MustParse("550e8400-e29b-41d4-a716-446655440008")).Times(1)
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(uuid.MustParse("550e8400-e29b-41d4-a716-446655440009")).Times(1)

				// Mock CreatePhraseEvaluationAndCriteriaScoreWithIsScoredState
				mockEvaluationScoresRepo.EXPECT().
					CreatePhraseEvaluationAndCriteriaScoreWithIsScoredState(ctx, mock.AnythingOfType("*repositories.CreatePhraseEvaluationAndCriteriaScoreWithIsScoredStateReqTx")).
					Return(nil)

				// Mock Redis Delete
				mockRedisClient.EXPECT().
					Delete(ctx, redisKey).
					Return(nil)

				return mockRedisClient, mockEvaluationRubricsRepo, mockEvaluationScoresRepo, mockInterviewTurnsRepo, mockGenerator
			},
			verify: func(t *testing.T, gotErr error) {
				assert.NoError(t, gotErr)
			},
		},
		{
			name:  "Error WithGetRubricWithCriteriaByNameError",
			input: validReq,
			setup: func() (*mockDatabase.MockRedisClient, *mockRepositories.MockEvaluationRubricsRepository, *mockRepositories.MockEvaluationScoresRepository, *mockRepositories.MockInterviewTurnsRepository, *mockUtils.MockGenerator) {
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockEvaluationRubricsRepo := new(mockRepositories.MockEvaluationRubricsRepository)
				mockEvaluationScoresRepo := new(mockRepositories.MockEvaluationScoresRepository)
				mockInterviewTurnsRepo := new(mockRepositories.MockInterviewTurnsRepository)
				mockGenerator := new(mockUtils.MockGenerator)

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s:%s", constants.RedisPrefixEvaluationRubric, "technical")).
					Return("", errors.New("cache error"))

				mockEvaluationRubricsRepo.EXPECT().
					GetRubricWithCriteriaByName(ctx, mock.AnythingOfType("*db.GetRubricWithCriteriaByNameParams")).
					Return(nil, errors.New("database error"))

				return mockRedisClient, mockEvaluationRubricsRepo, mockEvaluationScoresRepo, mockInterviewTurnsRepo, mockGenerator
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Equal(t, errors.New("database error"), gotErr)
			},
		},
		{
			name: "Error - Invalid Session ID",
			input: &entities.CalculateTurnScoreReq{
				SessionID:          "invalid-uuid",
				UserTurnID:         validUserTurnID,
				UserID:             validUserID,
				UserMessage:        "Test user message",
				InterviewerMessage: "Test interviewer message",
				CurrentState:       "technical",
				CurrentStateID:     "550e8400-e29b-41d4-a716-446655440007",
			},
			setup: func() (*mockDatabase.MockRedisClient, *mockRepositories.MockEvaluationRubricsRepository, *mockRepositories.MockEvaluationScoresRepository, *mockRepositories.MockInterviewTurnsRepository, *mockUtils.MockGenerator) {
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockEvaluationRubricsRepo := new(mockRepositories.MockEvaluationRubricsRepository)
				mockEvaluationScoresRepo := new(mockRepositories.MockEvaluationScoresRepository)
				mockInterviewTurnsRepo := new(mockRepositories.MockInterviewTurnsRepository)
				mockGenerator := new(mockUtils.MockGenerator)

				// Mock GetRubricWithCriteriaByName (called internally) - cache hit
				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s:%s", constants.RedisPrefixEvaluationRubric, "technical")).
					Return(string(jsonValidRubric), nil)

				// Mock InterviewFeedbackAndScore
				mockEvaluationScoresRepo.EXPECT().
					InterviewFeedbackAndScore(ctx, mock.AnythingOfType("*repositories.InterviewFeedbackAndScoreReq")).
					Return(validInterviewFeedbackResp, nil)

				// Mock generator calls - only the first one will be called before validation fails
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(uuid.MustParse(validEvaluationID)).Times(1)

				return mockRedisClient, mockEvaluationRubricsRepo, mockEvaluationScoresRepo, mockInterviewTurnsRepo, mockGenerator
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Equal(t, app_error.ErrCodeGeneralInvalidUUID, gotErr.(*app_error.AppError).Code)
			},
		},
		{
			name: "Error - Invalid User Turn ID",
			input: &entities.CalculateTurnScoreReq{
				SessionID:          validSessionID,
				UserTurnID:         "invalid-uuid",
				UserID:             validUserID,
				UserMessage:        "Test user message",
				InterviewerMessage: "Test interviewer message",
				CurrentState:       "technical",
				CurrentStateID:     "550e8400-e29b-41d4-a716-446655440007",
			},
			setup: func() (*mockDatabase.MockRedisClient, *mockRepositories.MockEvaluationRubricsRepository, *mockRepositories.MockEvaluationScoresRepository, *mockRepositories.MockInterviewTurnsRepository, *mockUtils.MockGenerator) {
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockEvaluationRubricsRepo := new(mockRepositories.MockEvaluationRubricsRepository)
				mockEvaluationScoresRepo := new(mockRepositories.MockEvaluationScoresRepository)
				mockInterviewTurnsRepo := new(mockRepositories.MockInterviewTurnsRepository)
				mockGenerator := new(mockUtils.MockGenerator)

				// Mock GetRubricWithCriteriaByName (called internally) - cache hit
				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s:%s", constants.RedisPrefixEvaluationRubric, "technical")).
					Return(string(jsonValidRubric), nil)

				// Mock InterviewFeedbackAndScore
				mockEvaluationScoresRepo.EXPECT().
					InterviewFeedbackAndScore(ctx, mock.AnythingOfType("*repositories.InterviewFeedbackAndScoreReq")).
					Return(validInterviewFeedbackResp, nil)

				// Mock generator calls - only the first one will be called before validation fails
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(uuid.MustParse(validEvaluationID)).Times(1)

				return mockRedisClient, mockEvaluationRubricsRepo, mockEvaluationScoresRepo, mockInterviewTurnsRepo, mockGenerator
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Equal(t, app_error.ErrCodeGeneralInvalidUUID, gotErr.(*app_error.AppError).Code)
			},
		},
		{
			name: "Error - Invalid User ID",
			input: &entities.CalculateTurnScoreReq{
				SessionID:          validSessionID,
				UserTurnID:         validUserTurnID,
				UserID:             "invalid-uuid",
				UserMessage:        "Test user message",
				InterviewerMessage: "Test interviewer message",
				CurrentState:       "technical",
				CurrentStateID:     "550e8400-e29b-41d4-a716-446655440007",
			},
			setup: func() (*mockDatabase.MockRedisClient, *mockRepositories.MockEvaluationRubricsRepository, *mockRepositories.MockEvaluationScoresRepository, *mockRepositories.MockInterviewTurnsRepository, *mockUtils.MockGenerator) {
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockEvaluationRubricsRepo := new(mockRepositories.MockEvaluationRubricsRepository)
				mockEvaluationScoresRepo := new(mockRepositories.MockEvaluationScoresRepository)
				mockInterviewTurnsRepo := new(mockRepositories.MockInterviewTurnsRepository)
				mockGenerator := new(mockUtils.MockGenerator)

				// Mock GetRubricWithCriteriaByName (called internally) - cache hit
				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s:%s", constants.RedisPrefixEvaluationRubric, "technical")).
					Return(string(jsonValidRubric), nil)

				// Mock InterviewFeedbackAndScore
				mockEvaluationScoresRepo.EXPECT().
					InterviewFeedbackAndScore(ctx, mock.AnythingOfType("*repositories.InterviewFeedbackAndScoreReq")).
					Return(validInterviewFeedbackResp, nil)

				// Mock generator calls - only the first one will be called before validation fails
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(uuid.MustParse(validEvaluationID)).Times(1)

				return mockRedisClient, mockEvaluationRubricsRepo, mockEvaluationScoresRepo, mockInterviewTurnsRepo, mockGenerator
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Equal(t, app_error.ErrCodeGeneralInvalidUUID, gotErr.(*app_error.AppError).Code)
			},
		},
		{
			name:  "Error WithInvalidCriteriaID",
			input: validReq,
			setup: func() (*mockDatabase.MockRedisClient, *mockRepositories.MockEvaluationRubricsRepository, *mockRepositories.MockEvaluationScoresRepository, *mockRepositories.MockInterviewTurnsRepository, *mockUtils.MockGenerator) {
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockEvaluationRubricsRepo := new(mockRepositories.MockEvaluationRubricsRepository)
				mockEvaluationScoresRepo := new(mockRepositories.MockEvaluationScoresRepository)
				mockInterviewTurnsRepo := new(mockRepositories.MockInterviewTurnsRepository)
				mockGenerator := new(mockUtils.MockGenerator)

				// Mock GetRubricWithCriteriaByName (called internally) - cache hit
				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s:%s", constants.RedisPrefixEvaluationRubric, validReq.CurrentState)).
					Return(string(jsonValidRubric), nil)

				// Mock InterviewFeedbackAndScore
				mockEvaluationScoresRepo.EXPECT().
					InterviewFeedbackAndScore(ctx, mock.AnythingOfType("*repositories.InterviewFeedbackAndScoreReq")).
					Return(invalidCriterionResp, nil)

				// Mock generator calls - only the first one will be called before validation fails
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(uuid.MustParse(validEvaluationID)).Times(1)

				return mockRedisClient, mockEvaluationRubricsRepo, mockEvaluationScoresRepo, mockInterviewTurnsRepo, mockGenerator
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Equal(t, app_error.ErrCodeGeneralInvalidUUID, gotErr.(*app_error.AppError).Code)
			},
		},

		{
			name:  "Error - InterviewFeedbackAndScore Failed",
			input: validReq,
			setup: func() (*mockDatabase.MockRedisClient, *mockRepositories.MockEvaluationRubricsRepository, *mockRepositories.MockEvaluationScoresRepository, *mockRepositories.MockInterviewTurnsRepository, *mockUtils.MockGenerator) {
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockEvaluationRubricsRepo := new(mockRepositories.MockEvaluationRubricsRepository)
				mockEvaluationScoresRepo := new(mockRepositories.MockEvaluationScoresRepository)
				mockInterviewTurnsRepo := new(mockRepositories.MockInterviewTurnsRepository)
				mockGenerator := new(mockUtils.MockGenerator)

				// Mock GetRubricWithCriteriaByName (called internally)
				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s:%s", constants.RedisPrefixEvaluationRubric, validReq.CurrentState)).
					Return(string(jsonValidRubric), nil)

				// Mock InterviewFeedbackAndScore failure
				mockEvaluationScoresRepo.EXPECT().
					InterviewFeedbackAndScore(ctx, mock.AnythingOfType("*repositories.InterviewFeedbackAndScoreReq")).
					Return(nil, errors.New("LLM service error"))

				return mockRedisClient, mockEvaluationRubricsRepo, mockEvaluationScoresRepo, mockInterviewTurnsRepo, mockGenerator
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Equal(t, "LLM service error", gotErr.Error())
			},
		},
		{
			name:  "Error WithInvalidRubricID",
			input: validReq,
			setup: func() (*mockDatabase.MockRedisClient, *mockRepositories.MockEvaluationRubricsRepository, *mockRepositories.MockEvaluationScoresRepository, *mockRepositories.MockInterviewTurnsRepository, *mockUtils.MockGenerator) {
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockEvaluationRubricsRepo := new(mockRepositories.MockEvaluationRubricsRepository)
				mockEvaluationScoresRepo := new(mockRepositories.MockEvaluationScoresRepository)
				mockInterviewTurnsRepo := new(mockRepositories.MockInterviewTurnsRepository)
				mockGenerator := new(mockUtils.MockGenerator)

				// Mock GetRubricWithCriteriaByName (called internally)
				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s:%s", constants.RedisPrefixEvaluationRubric, validReq.CurrentState)).
					Return(string(jsonInvalidRubric), nil)

				// Mock InterviewFeedbackAndScore
				mockEvaluationScoresRepo.EXPECT().
					InterviewFeedbackAndScore(ctx, mock.AnythingOfType("*repositories.InterviewFeedbackAndScoreReq")).
					Return(validInterviewFeedbackResp, nil)

				// Mock generator calls
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(uuid.MustParse(validEvaluationID)).Times(1)

				return mockRedisClient, mockEvaluationRubricsRepo, mockEvaluationScoresRepo, mockInterviewTurnsRepo, mockGenerator
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Equal(t, app_error.ErrCodeGeneralInvalidUUID, gotErr.(*app_error.AppError).Code)
			},
		},
		{
			name:  "Error WithInvalidCriterionID",
			input: validReq,
			setup: func() (*mockDatabase.MockRedisClient, *mockRepositories.MockEvaluationRubricsRepository, *mockRepositories.MockEvaluationScoresRepository, *mockRepositories.MockInterviewTurnsRepository, *mockUtils.MockGenerator) {
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockEvaluationRubricsRepo := new(mockRepositories.MockEvaluationRubricsRepository)
				mockEvaluationScoresRepo := new(mockRepositories.MockEvaluationScoresRepository)
				mockInterviewTurnsRepo := new(mockRepositories.MockInterviewTurnsRepository)
				mockGenerator := new(mockUtils.MockGenerator)

				// Mock GetRubricWithCriteriaByName (called internally) - cache hit
				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s:%s", constants.RedisPrefixEvaluationRubric, validReq.CurrentState)).
					Return(string(jsonValidRubric), nil)

				// Mock InterviewFeedbackAndScore
				mockEvaluationScoresRepo.EXPECT().
					InterviewFeedbackAndScore(ctx, mock.AnythingOfType("*repositories.InterviewFeedbackAndScoreReq")).
					Return(invalidInterviewFeedbackResp, nil)

				// Mock generator calls - need to match the order they're called
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(uuid.MustParse(validEvaluationID)).Times(1)

				return mockRedisClient, mockEvaluationRubricsRepo, mockEvaluationScoresRepo, mockInterviewTurnsRepo, mockGenerator
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Equal(t, app_error.ErrCodeGeneralInvalidUUID, gotErr.(*app_error.AppError).Code)
			},
		},
		{
			name:  "Error - CreateEvaluationWithCriteriaScoreAndImproveSentenceTxFailed",
			input: validReq,
			setup: func() (*mockDatabase.MockRedisClient, *mockRepositories.MockEvaluationRubricsRepository, *mockRepositories.MockEvaluationScoresRepository, *mockRepositories.MockInterviewTurnsRepository, *mockUtils.MockGenerator) {
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockEvaluationRubricsRepo := new(mockRepositories.MockEvaluationRubricsRepository)
				mockEvaluationScoresRepo := new(mockRepositories.MockEvaluationScoresRepository)
				mockInterviewTurnsRepo := new(mockRepositories.MockInterviewTurnsRepository)
				mockGenerator := new(mockUtils.MockGenerator)

				// Mock GetRubricWithCriteriaByName (called internally)
				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s:%s", constants.RedisPrefixEvaluationRubric, validReq.CurrentState)).
					Return(string(jsonValidRubric), nil)

				// Mock InterviewFeedbackAndScore
				mockEvaluationScoresRepo.EXPECT().
					InterviewFeedbackAndScore(ctx, mock.AnythingOfType("*repositories.InterviewFeedbackAndScoreReq")).
					Return(validInterviewFeedbackResp, nil)

				// Mock generator calls
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(uuid.MustParse(validEvaluationID)).Times(1)
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(uuid.MustParse(validCriterionID)).Times(1)
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(uuid.MustParse(validImproveSentenceID)).Times(1)

				// Mock CreateEvaluationWithCriteriaScoreAndImproveSentenceTx failure
				mockEvaluationScoresRepo.EXPECT().
					CreateEvaluationWithCriteriaScoreAndImproveSentenceTx(ctx, mock.AnythingOfType("*repositories.CreateEvaluationAndScoreTxReq")).
					Return(errors.New("database error"))

				return mockRedisClient, mockEvaluationRubricsRepo, mockEvaluationScoresRepo, mockInterviewTurnsRepo, mockGenerator
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Equal(t, "database error", gotErr.Error())
			},
		},
		{
			name:  "Success - Last Turn ID Does Not Match",
			input: validReq,
			setup: func() (*mockDatabase.MockRedisClient, *mockRepositories.MockEvaluationRubricsRepository, *mockRepositories.MockEvaluationScoresRepository, *mockRepositories.MockInterviewTurnsRepository, *mockUtils.MockGenerator) {
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockEvaluationRubricsRepo := new(mockRepositories.MockEvaluationRubricsRepository)
				mockEvaluationScoresRepo := new(mockRepositories.MockEvaluationScoresRepository)
				mockInterviewTurnsRepo := new(mockRepositories.MockInterviewTurnsRepository)
				mockGenerator := new(mockUtils.MockGenerator)

				// Mock GetRubricWithCriteriaByName (called internally) - cache hit
				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s:%s", constants.RedisPrefixEvaluationRubric, validReq.CurrentState)).
					Return(string(jsonValidRubric), nil)

				// Mock InterviewFeedbackAndScore
				mockEvaluationScoresRepo.EXPECT().
					InterviewFeedbackAndScore(ctx, mock.AnythingOfType("*repositories.InterviewFeedbackAndScoreReq")).
					Return(validInterviewFeedbackResp, nil)

				// Mock generator calls - need to match the order they're called
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(uuid.MustParse(validEvaluationID)).Times(1)
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(uuid.MustParse(validCriterionID)).Times(1)
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(uuid.MustParse(validImproveSentenceID)).Times(1)

				// Mock CreateEvaluationWithCriteriaScoreAndImproveSentenceTx
				mockEvaluationScoresRepo.EXPECT().
					CreateEvaluationWithCriteriaScoreAndImproveSentenceTx(ctx, mock.AnythingOfType("*repositories.CreateEvaluationAndScoreTxReq")).
					Return(nil)

				// Mock FlagIsScoreEvaluated
				mockInterviewTurnsRepo.EXPECT().
					FlagIsScoreEvaluated(ctx, uuid.MustParse(validUserTurnID)).
					Return(nil)

				// Mock Redis Get for last turn ID (with retries) - return different turn ID
				redisKey := fmt.Sprintf("%s%s:%s", constants.RedisPrefixInterviewLastTurnID, validReq.SessionID, validReq.CurrentState)
				mockRedisClient.EXPECT().
					Get(ctx, redisKey).
					Return("550e8400-e29b-41d4-a716-446655440010", nil) // Different turn ID

				// Mock Redis Delete
				mockRedisClient.EXPECT().
					Delete(ctx, redisKey).
					Return(nil)

				return mockRedisClient, mockEvaluationRubricsRepo, mockEvaluationScoresRepo, mockInterviewTurnsRepo, mockGenerator
			},
			verify: func(t *testing.T, gotErr error) {
				assert.NoError(t, gotErr)
			},
		},
		{
			name:  "Success - Redis Get Retries",
			input: validReq,
			setup: func() (*mockDatabase.MockRedisClient, *mockRepositories.MockEvaluationRubricsRepository, *mockRepositories.MockEvaluationScoresRepository, *mockRepositories.MockInterviewTurnsRepository, *mockUtils.MockGenerator) {
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockEvaluationRubricsRepo := new(mockRepositories.MockEvaluationRubricsRepository)
				mockEvaluationScoresRepo := new(mockRepositories.MockEvaluationScoresRepository)
				mockInterviewTurnsRepo := new(mockRepositories.MockInterviewTurnsRepository)
				mockGenerator := new(mockUtils.MockGenerator)

				// Mock GetRubricWithCriteriaByName (called internally) - cache hit
				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s:%s", constants.RedisPrefixEvaluationRubric, validReq.CurrentState)).
					Return(string(jsonValidRubric), nil)

				// Mock InterviewFeedbackAndScore
				mockEvaluationScoresRepo.EXPECT().
					InterviewFeedbackAndScore(ctx, mock.AnythingOfType("*repositories.InterviewFeedbackAndScoreReq")).
					Return(validInterviewFeedbackResp, nil)

				// Mock generator calls - need to match the order they're called
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(uuid.MustParse(validEvaluationID)).Times(1)
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(uuid.MustParse(validCriterionID)).Times(1)
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(uuid.MustParse(validImproveSentenceID)).Times(1)

				// Mock CreateEvaluationWithCriteriaScoreAndImproveSentenceTx
				mockEvaluationScoresRepo.EXPECT().
					CreateEvaluationWithCriteriaScoreAndImproveSentenceTx(ctx, mock.AnythingOfType("*repositories.CreateEvaluationAndScoreTxReq")).
					Return(nil)

				// Mock FlagIsScoreEvaluated
				mockInterviewTurnsRepo.EXPECT().
					FlagIsScoreEvaluated(ctx, uuid.MustParse(validUserTurnID)).
					Return(nil)

				// Mock Redis Get for last turn ID with retries - first two fail, third succeeds
				redisKey := fmt.Sprintf("%s%s:%s", constants.RedisPrefixInterviewLastTurnID, validReq.SessionID, validReq.CurrentState)
				mockRedisClient.EXPECT().
					Get(ctx, redisKey).
					Return("", redis.Nil).
					Times(2)
				mockRedisClient.EXPECT().
					Get(ctx, redisKey).
					Return(validUserTurnID, nil)

				// Mock CalculateEvaluationInOldState (called when last turn ID matches)
				mockEvaluationScoresRepo.EXPECT().
					GetEvaluationSummaryJsonBySessionAndState(ctx, mock.AnythingOfType("*db.GetEvaluationSummaryJsonBySessionAndStateParams")).
					Return([]byte(`{"criteria":[],"evaluation_avg_score":8.5}`), nil)

				mockEvaluationScoresRepo.EXPECT().
					CalculateEachCriteriaCommentBySessionAndState(ctx, mock.AnythingOfType("*repositories.PreProcessedCriteriaReq")).
					Return(&repositories.PostProcessedCriteriaResp{
						Criteria: []repositories.PostProcessedCriteria{
							{
								CriteriaID:       validCriterionID,
								CriteriaName:     "Problem Solving",
								CriteriaAvgScore: 8.0,
								CriteriaComment:  "Good problem solving skills",
							},
						},
					}, nil)

				// Mock generator calls for CalculateEvaluationInOldState
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(uuid.MustParse("550e8400-e29b-41d4-a716-446655440008")).Times(1)
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(uuid.MustParse("550e8400-e29b-41d4-a716-446655440009")).Times(1)

				// Mock CreatePhraseEvaluationAndCriteriaScoreWithIsScoredState
				mockEvaluationScoresRepo.EXPECT().
					CreatePhraseEvaluationAndCriteriaScoreWithIsScoredState(ctx, mock.AnythingOfType("*repositories.CreatePhraseEvaluationAndCriteriaScoreWithIsScoredStateReqTx")).
					Return(nil)

				// Mock Redis Delete
				mockRedisClient.EXPECT().
					Delete(ctx, redisKey).
					Return(nil)

				return mockRedisClient, mockEvaluationRubricsRepo, mockEvaluationScoresRepo, mockInterviewTurnsRepo, mockGenerator
			},
			verify: func(t *testing.T, gotErr error) {
				assert.NoError(t, gotErr)
			},
		},
		{
			name:  "Success - Redis Get Returns Empty After Retries",
			input: validReq,
			setup: func() (*mockDatabase.MockRedisClient, *mockRepositories.MockEvaluationRubricsRepository, *mockRepositories.MockEvaluationScoresRepository, *mockRepositories.MockInterviewTurnsRepository, *mockUtils.MockGenerator) {
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockEvaluationRubricsRepo := new(mockRepositories.MockEvaluationRubricsRepository)
				mockEvaluationScoresRepo := new(mockRepositories.MockEvaluationScoresRepository)
				mockInterviewTurnsRepo := new(mockRepositories.MockInterviewTurnsRepository)
				mockGenerator := new(mockUtils.MockGenerator)

				// Mock GetRubricWithCriteriaByName (called internally) - cache hit
				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s:%s", constants.RedisPrefixEvaluationRubric, validReq.CurrentState)).
					Return(string(jsonValidRubric), nil)

				// Mock InterviewFeedbackAndScore
				mockEvaluationScoresRepo.EXPECT().
					InterviewFeedbackAndScore(ctx, mock.AnythingOfType("*repositories.InterviewFeedbackAndScoreReq")).
					Return(validInterviewFeedbackResp, nil)

				// Mock generator calls - need to match the order they're called
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(uuid.MustParse(validEvaluationID)).Times(1)
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(uuid.MustParse(validCriterionID)).Times(1)
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(uuid.MustParse(validImproveSentenceID)).Times(1)

				// Mock CreateEvaluationWithCriteriaScoreAndImproveSentenceTx
				mockEvaluationScoresRepo.EXPECT().
					CreateEvaluationWithCriteriaScoreAndImproveSentenceTx(ctx, mock.AnythingOfType("*repositories.CreateEvaluationAndScoreTxReq")).
					Return(nil)

				// Mock FlagIsScoreEvaluated
				mockInterviewTurnsRepo.EXPECT().
					FlagIsScoreEvaluated(ctx, uuid.MustParse(validUserTurnID)).
					Return(nil)

				// Mock Redis Get for last turn ID with retries - all fail
				redisKey := fmt.Sprintf("%s%s:%s", constants.RedisPrefixInterviewLastTurnID, validReq.SessionID, validReq.CurrentState)
				mockRedisClient.EXPECT().
					Get(ctx, redisKey).
					Return("", redis.Nil).
					Times(3)

				return mockRedisClient, mockEvaluationRubricsRepo, mockEvaluationScoresRepo, mockInterviewTurnsRepo, mockGenerator
			},
			verify: func(t *testing.T, gotErr error) {
				assert.NoError(t, gotErr)
			},
		},
		{
			name:  "Error - FlagIsScoreEvaluated Failed",
			input: validReq,
			setup: func() (*mockDatabase.MockRedisClient, *mockRepositories.MockEvaluationRubricsRepository, *mockRepositories.MockEvaluationScoresRepository, *mockRepositories.MockInterviewTurnsRepository, *mockUtils.MockGenerator) {
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockEvaluationRubricsRepo := new(mockRepositories.MockEvaluationRubricsRepository)
				mockEvaluationScoresRepo := new(mockRepositories.MockEvaluationScoresRepository)
				mockInterviewTurnsRepo := new(mockRepositories.MockInterviewTurnsRepository)
				mockGenerator := new(mockUtils.MockGenerator)

				// Mock GetRubricWithCriteriaByName (called internally) - cache hit
				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s:%s", constants.RedisPrefixEvaluationRubric, validReq.CurrentState)).
					Return(string(jsonValidRubric), nil)

				// Mock InterviewFeedbackAndScore
				mockEvaluationScoresRepo.EXPECT().
					InterviewFeedbackAndScore(ctx, mock.AnythingOfType("*repositories.InterviewFeedbackAndScoreReq")).
					Return(validInterviewFeedbackResp, nil)

				// Mock generator calls - need to match the order they're called
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(uuid.MustParse(validEvaluationID)).Times(1)
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(uuid.MustParse(validCriterionID)).Times(1)
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(uuid.MustParse(validImproveSentenceID)).Times(1)

				// Mock CreateEvaluationWithCriteriaScoreAndImproveSentenceTx
				mockEvaluationScoresRepo.EXPECT().
					CreateEvaluationWithCriteriaScoreAndImproveSentenceTx(ctx, mock.AnythingOfType("*repositories.CreateEvaluationAndScoreTxReq")).
					Return(nil)

				// Mock FlagIsScoreEvaluated failure
				mockInterviewTurnsRepo.EXPECT().
					FlagIsScoreEvaluated(ctx, uuid.MustParse(validUserTurnID)).
					Return(errors.New("database error"))

				return mockRedisClient, mockEvaluationRubricsRepo, mockEvaluationScoresRepo, mockInterviewTurnsRepo, mockGenerator
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Equal(t, "database error", gotErr.Error())
			},
		},
		{
			name:  "Error - Redis Get Failed After Retries",
			input: validReq,
			setup: func() (*mockDatabase.MockRedisClient, *mockRepositories.MockEvaluationRubricsRepository, *mockRepositories.MockEvaluationScoresRepository, *mockRepositories.MockInterviewTurnsRepository, *mockUtils.MockGenerator) {
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockEvaluationRubricsRepo := new(mockRepositories.MockEvaluationRubricsRepository)
				mockEvaluationScoresRepo := new(mockRepositories.MockEvaluationScoresRepository)
				mockInterviewTurnsRepo := new(mockRepositories.MockInterviewTurnsRepository)
				mockGenerator := new(mockUtils.MockGenerator)

				// Mock GetRubricWithCriteriaByName (called internally) - cache hit
				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s:%s", constants.RedisPrefixEvaluationRubric, validReq.CurrentState)).
					Return(string(jsonValidRubric), nil)

				// Mock InterviewFeedbackAndScore
				mockEvaluationScoresRepo.EXPECT().
					InterviewFeedbackAndScore(ctx, mock.AnythingOfType("*repositories.InterviewFeedbackAndScoreReq")).
					Return(validInterviewFeedbackResp, nil)

				// Mock generator calls - need to match the order they're called
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(uuid.MustParse(validEvaluationID)).Times(1)
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(uuid.MustParse(validCriterionID)).Times(1)
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(uuid.MustParse(validImproveSentenceID)).Times(1)

				// Mock CreateEvaluationWithCriteriaScoreAndImproveSentenceTx
				mockEvaluationScoresRepo.EXPECT().
					CreateEvaluationWithCriteriaScoreAndImproveSentenceTx(ctx, mock.AnythingOfType("*repositories.CreateEvaluationAndScoreTxReq")).
					Return(nil)

				// Mock FlagIsScoreEvaluated
				mockInterviewTurnsRepo.EXPECT().
					FlagIsScoreEvaluated(ctx, uuid.MustParse(validUserTurnID)).
					Return(nil)

				// Mock Redis Get for last turn ID with retries - all fail with non-Nil error
				redisKey := fmt.Sprintf("%s%s:%s", constants.RedisPrefixInterviewLastTurnID, validReq.SessionID, validReq.CurrentState)
				mockRedisClient.EXPECT().
					Get(ctx, redisKey).
					Return("", errors.New("redis error"))

				return mockRedisClient, mockEvaluationRubricsRepo, mockEvaluationScoresRepo, mockInterviewTurnsRepo, mockGenerator
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Equal(t, "redis error", gotErr.Error())
			},
		},
		{
			name:  "Error - Redis Delete Failed",
			input: validReq,
			setup: func() (*mockDatabase.MockRedisClient, *mockRepositories.MockEvaluationRubricsRepository, *mockRepositories.MockEvaluationScoresRepository, *mockRepositories.MockInterviewTurnsRepository, *mockUtils.MockGenerator) {
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockEvaluationRubricsRepo := new(mockRepositories.MockEvaluationRubricsRepository)
				mockEvaluationScoresRepo := new(mockRepositories.MockEvaluationScoresRepository)
				mockInterviewTurnsRepo := new(mockRepositories.MockInterviewTurnsRepository)
				mockGenerator := new(mockUtils.MockGenerator)

				// Mock GetRubricWithCriteriaByName (called internally) - cache hit
				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s:%s", constants.RedisPrefixEvaluationRubric, validReq.CurrentState)).
					Return(string(jsonValidRubric), nil)

				// Mock InterviewFeedbackAndScore
				mockEvaluationScoresRepo.EXPECT().
					InterviewFeedbackAndScore(ctx, mock.AnythingOfType("*repositories.InterviewFeedbackAndScoreReq")).
					Return(validInterviewFeedbackResp, nil)

				// Mock generator calls - need to match the order they're called
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(uuid.MustParse(validEvaluationID)).Times(1)
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(uuid.MustParse(validCriterionID)).Times(1)
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(uuid.MustParse(validImproveSentenceID)).Times(1)

				// Mock CreateEvaluationWithCriteriaScoreAndImproveSentenceTx
				mockEvaluationScoresRepo.EXPECT().
					CreateEvaluationWithCriteriaScoreAndImproveSentenceTx(ctx, mock.AnythingOfType("*repositories.CreateEvaluationAndScoreTxReq")).
					Return(nil)

				// Mock FlagIsScoreEvaluated
				mockInterviewTurnsRepo.EXPECT().
					FlagIsScoreEvaluated(ctx, uuid.MustParse(validUserTurnID)).
					Return(nil)

				// Mock Redis Get for last turn ID
				redisKey := fmt.Sprintf("%s%s:%s", constants.RedisPrefixInterviewLastTurnID, validReq.SessionID, validReq.CurrentState)
				mockRedisClient.EXPECT().
					Get(ctx, redisKey).
					Return(validUserTurnID, nil)

				// Mock CalculateEvaluationInOldState (called when last turn ID matches)
				mockEvaluationScoresRepo.EXPECT().
					GetEvaluationSummaryJsonBySessionAndState(ctx, mock.AnythingOfType("*db.GetEvaluationSummaryJsonBySessionAndStateParams")).
					Return([]byte(`{"criteria":[],"evaluation_avg_score":8.5}`), nil)

				mockEvaluationScoresRepo.EXPECT().
					CalculateEachCriteriaCommentBySessionAndState(ctx, mock.AnythingOfType("*repositories.PreProcessedCriteriaReq")).
					Return(&repositories.PostProcessedCriteriaResp{
						Criteria: []repositories.PostProcessedCriteria{
							{
								CriteriaID:       validCriterionID,
								CriteriaName:     "Problem Solving",
								CriteriaAvgScore: 8.0,
								CriteriaComment:  "Good problem solving skills",
							},
						},
					}, nil)

				// Mock generator calls for CalculateEvaluationInOldState
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(uuid.MustParse("550e8400-e29b-41d4-a716-446655440008")).Times(1)
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(uuid.MustParse("550e8400-e29b-41d4-a716-446655440009")).Times(1)

				// Mock CreatePhraseEvaluationAndCriteriaScoreWithIsScoredState
				mockEvaluationScoresRepo.EXPECT().
					CreatePhraseEvaluationAndCriteriaScoreWithIsScoredState(ctx, mock.AnythingOfType("*repositories.CreatePhraseEvaluationAndCriteriaScoreWithIsScoredStateReqTx")).
					Return(nil)

				// Mock Redis Delete failure
				mockRedisClient.EXPECT().
					Delete(ctx, redisKey).
					Return(errors.New("redis delete error"))

				return mockRedisClient, mockEvaluationRubricsRepo, mockEvaluationScoresRepo, mockInterviewTurnsRepo, mockGenerator
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Equal(t, "redis delete error", gotErr.Error())
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			mockRedisClient, mockEvaluationRubricsRepo, mockEvaluationScoresRepo, mockInterviewTurnsRepo, mockGenerator := tC.setup()

			defer func() {
				if mockRedisClient != nil {
					mockRedisClient.AssertExpectations(t)
				}
				if mockEvaluationRubricsRepo != nil {
					mockEvaluationRubricsRepo.AssertExpectations(t)
				}
				if mockEvaluationScoresRepo != nil {
					mockEvaluationScoresRepo.AssertExpectations(t)
				}
				if mockInterviewTurnsRepo != nil {
					mockInterviewTurnsRepo.AssertExpectations(t)
				}
				if mockGenerator != nil {
					mockGenerator.AssertExpectations(t)
				}
			}()

			svc := NewEvaluationService(
				lgr,
				mockRedisClient,
				mockEvaluationRubricsRepo,
				mockEvaluationScoresRepo,
				mockInterviewTurnsRepo,
				nil,
				nil,
				mockGenerator,
			)

			gotErr := svc.CalculateTurnScore(ctx, tC.input)

			tC.verify(t, gotErr)
		})
	}
}

func TestEvaluationService_CalculateEvaluationInOldState(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()

	validSessionID := "550e8400-e29b-41d4-a716-446655440000"
	validInterviewStateID := "550e8400-e29b-41d4-a716-446655440001"
	validEvaluationID := "550e8400-e29b-41d4-a716-446655440002"
	validCriteriaID1 := "550e8400-e29b-41d4-a716-446655440003"
	validCriteriaID2 := "550e8400-e29b-41d4-a716-446655440004"
	validScoreID1 := "550e8400-e29b-41d4-a716-446655440005"
	validScoreID2 := "550e8400-e29b-41d4-a716-446655440006"

	validReq := &entities.CalculateEvaluationInOldStateReq{
		SessionID:        validSessionID,
		CurrentState:     "technical",
		InterviewStateID: validInterviewStateID,
	}

	validEvaluationSummaryResp := &entities.GetEvaluationSummaryJsonBySessionAndStateResp{
		Criteria: []entities.EvaluationCriteria{
			{
				CriteriaID:   validCriteriaID1,
				CriteriaName: "Problem Solving",
				CriteriaScore: []entities.CriteriaScoreOld{
					{Score: 8, Comment: "Good approach"},
					{Score: 9, Comment: "Excellent solution"},
					{Score: 7, Comment: "Needs improvement"},
				},
			},
			{
				CriteriaID:   validCriteriaID2,
				CriteriaName: "Communication",
				CriteriaScore: []entities.CriteriaScoreOld{
					{Score: 6, Comment: "Could be clearer"},
					{Score: 8, Comment: "Well explained"},
				},
			},
		},
		EvaluationAvgScore: 7.4,
	}

	validPostProcessedCriteriaResp := &repositories.PostProcessedCriteriaResp{
		Criteria: []repositories.PostProcessedCriteria{
			{
				CriteriaID:       validCriteriaID1,
				CriteriaName:     "Problem Solving",
				CriteriaAvgScore: 8.0,
				CriteriaComment:  "Overall good problem solving skills with room for improvement in explanation",
			},
			{
				CriteriaID:       validCriteriaID2,
				CriteriaName:     "Communication",
				CriteriaAvgScore: 7.0,
				CriteriaComment:  "Communication skills are developing well",
			},
		},
	}

	inValidPostProcessedCriteriaResp := &repositories.PostProcessedCriteriaResp{
		Criteria: []repositories.PostProcessedCriteria{
			{
				CriteriaID:       "invalid-criteria-id",
				CriteriaName:     "Problem Solving",
				CriteriaAvgScore: 8.0,
				CriteriaComment:  "Overall good problem solving skills with room for improvement in explanation",
			},
			{
				CriteriaID:       "invalid-criteria-id",
				CriteriaName:     "Communication",
				CriteriaAvgScore: 7.0,
				CriteriaComment:  "Communication skills are developing well",
			},
		},
	}

	jsonEvaluationSummaryResp, err := json.Marshal(validEvaluationSummaryResp)
	if err != nil {
		t.Fatalf("Failed to marshal evaluation summary response: %v", err)
	}

	testCases := []struct {
		name   string
		input  *entities.CalculateEvaluationInOldStateReq
		setup  func() (*mockDatabase.MockRedisClient, *mockRepositories.MockEvaluationRubricsRepository, *mockRepositories.MockEvaluationScoresRepository, *mockRepositories.MockInterviewTurnsRepository, *mockUtils.MockGenerator)
		verify func(t *testing.T, gotErr error)
	}{
		{
			name:  "Success - Complete Flow",
			input: validReq,
			setup: func() (*mockDatabase.MockRedisClient, *mockRepositories.MockEvaluationRubricsRepository, *mockRepositories.MockEvaluationScoresRepository, *mockRepositories.MockInterviewTurnsRepository, *mockUtils.MockGenerator) {
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockEvaluationRubricsRepo := new(mockRepositories.MockEvaluationRubricsRepository)
				mockEvaluationScoresRepo := new(mockRepositories.MockEvaluationScoresRepository)
				mockInterviewTurnsRepo := new(mockRepositories.MockInterviewTurnsRepository)
				mockGenerator := new(mockUtils.MockGenerator)

				// Mock GetEvaluationSummaryJsonBySessionAndState
				mockEvaluationScoresRepo.EXPECT().
					GetEvaluationSummaryJsonBySessionAndState(ctx, mock.MatchedBy(func(req *db.GetEvaluationSummaryJsonBySessionAndStateParams) bool {
						return req.SessionID == uuid.MustParse(validSessionID) &&
							req.CurrentState == validReq.CurrentState
					})).
					Return(jsonEvaluationSummaryResp, nil)

				// Mock CalculateEachCriteriaCommentBySessionAndState
				mockEvaluationScoresRepo.EXPECT().
					CalculateEachCriteriaCommentBySessionAndState(ctx, mock.MatchedBy(func(req *repositories.PreProcessedCriteriaReq) bool {
						return req.Criteria[0].CriteriaID == validCriteriaID1 &&
							req.Criteria[1].CriteriaID == validCriteriaID2
					})).
					Return(validPostProcessedCriteriaResp, nil)

				// Mock generator calls
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(uuid.MustParse(validEvaluationID)).Times(1)
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(uuid.MustParse(validScoreID1)).Times(1)
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(uuid.MustParse(validScoreID2)).Times(1)

				// Mock CreatePhraseEvaluationAndCriteriaScoreWithIsScoredState
				mockEvaluationScoresRepo.EXPECT().
					CreatePhraseEvaluationAndCriteriaScoreWithIsScoredState(ctx, mock.MatchedBy(func(req *repositories.CreatePhraseEvaluationAndCriteriaScoreWithIsScoredStateReqTx) bool {
						return req.SessionID == uuid.MustParse(validSessionID) &&
							req.StateID == uuid.MustParse(validInterviewStateID) &&
							req.StateName == validReq.CurrentState &&
							req.OverallScore == validEvaluationSummaryResp.EvaluationAvgScore &&
							req.Criteria.CriteriaID[0] == uuid.MustParse(validCriteriaID1) &&
							req.Criteria.CriteriaID[1] == uuid.MustParse(validCriteriaID2) &&
							req.Criteria.CriteriaName[0] == "Problem Solving" &&
							req.Criteria.CriteriaName[1] == "Communication" &&
							req.Criteria.CriteriaAvgScore[0] == 8.0 &&
							req.Criteria.CriteriaAvgScore[1] == 7.0
					})).
					Return(nil)

				return mockRedisClient, mockEvaluationRubricsRepo, mockEvaluationScoresRepo, mockInterviewTurnsRepo, mockGenerator
			},
			verify: func(t *testing.T, gotErr error) {
				assert.NoError(t, gotErr)
			},
		},
		{
			name:  "Success - Redis Miss, DB Check Success",
			input: validReq,
			setup: func() (*mockDatabase.MockRedisClient, *mockRepositories.MockEvaluationRubricsRepository, *mockRepositories.MockEvaluationScoresRepository, *mockRepositories.MockInterviewTurnsRepository, *mockUtils.MockGenerator) {
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockEvaluationRubricsRepo := new(mockRepositories.MockEvaluationRubricsRepository)
				mockEvaluationScoresRepo := new(mockRepositories.MockEvaluationScoresRepository)
				mockInterviewTurnsRepo := new(mockRepositories.MockInterviewTurnsRepository)
				mockGenerator := new(mockUtils.MockGenerator)

				// Mock GetEvaluationSummaryJsonBySessionAndState
				mockEvaluationScoresRepo.EXPECT().
					GetEvaluationSummaryJsonBySessionAndState(ctx, mock.MatchedBy(func(req *db.GetEvaluationSummaryJsonBySessionAndStateParams) bool {
						return req.SessionID == uuid.MustParse(validSessionID) &&
							req.CurrentState == validReq.CurrentState
					})).
					Return(jsonEvaluationSummaryResp, nil)

				// Mock CalculateEachCriteriaCommentBySessionAndState
				mockEvaluationScoresRepo.EXPECT().
					CalculateEachCriteriaCommentBySessionAndState(ctx, mock.MatchedBy(func(req *repositories.PreProcessedCriteriaReq) bool {
						return req.Criteria[0].CriteriaID == validCriteriaID1 &&
							req.Criteria[1].CriteriaID == validCriteriaID2
					})).
					Return(validPostProcessedCriteriaResp, nil)

				// Mock generator calls
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(uuid.MustParse(validEvaluationID)).Times(1)
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(uuid.MustParse(validScoreID1)).Times(1)
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(uuid.MustParse(validScoreID2)).Times(1)

				// Mock CreatePhraseEvaluationAndCriteriaScoreWithIsScoredState
				mockEvaluationScoresRepo.EXPECT().
					CreatePhraseEvaluationAndCriteriaScoreWithIsScoredState(ctx, mock.MatchedBy(func(req *repositories.CreatePhraseEvaluationAndCriteriaScoreWithIsScoredStateReqTx) bool {
						return req.SessionID == uuid.MustParse(validSessionID) &&
							req.StateID == uuid.MustParse(validInterviewStateID) &&
							req.StateName == validReq.CurrentState &&
							req.OverallScore == validEvaluationSummaryResp.EvaluationAvgScore &&
							req.Criteria.CriteriaID[0] == uuid.MustParse(validCriteriaID1) &&
							req.Criteria.CriteriaID[1] == uuid.MustParse(validCriteriaID2) &&
							req.Criteria.CriteriaName[0] == "Problem Solving" &&
							req.Criteria.CriteriaName[1] == "Communication" &&
							req.Criteria.CriteriaAvgScore[0] == 8.0 &&
							req.Criteria.CriteriaAvgScore[1] == 7.0
					})).
					Return(nil)

				return mockRedisClient, mockEvaluationRubricsRepo, mockEvaluationScoresRepo, mockInterviewTurnsRepo, mockGenerator
			},
			verify: func(t *testing.T, gotErr error) {
				assert.NoError(t, gotErr)
			},
		},
		{
			name: "Error - Invalid Session ID",
			input: &entities.CalculateEvaluationInOldStateReq{
				SessionID:        "invalid-uuid",
				CurrentState:     "technical",
				InterviewStateID: validInterviewStateID,
			},
			setup: func() (*mockDatabase.MockRedisClient, *mockRepositories.MockEvaluationRubricsRepository, *mockRepositories.MockEvaluationScoresRepository, *mockRepositories.MockInterviewTurnsRepository, *mockUtils.MockGenerator) {
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockEvaluationRubricsRepo := new(mockRepositories.MockEvaluationRubricsRepository)
				mockEvaluationScoresRepo := new(mockRepositories.MockEvaluationScoresRepository)
				mockInterviewTurnsRepo := new(mockRepositories.MockInterviewTurnsRepository)
				mockGenerator := new(mockUtils.MockGenerator)

				return mockRedisClient, mockEvaluationRubricsRepo, mockEvaluationScoresRepo, mockInterviewTurnsRepo, mockGenerator
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Equal(t, app_error.ErrCodeGeneralInvalidUUID, gotErr.(*app_error.AppError).Code)
			},
		},
		{
			name: "Error - Invalid Interview State ID",
			input: &entities.CalculateEvaluationInOldStateReq{
				SessionID:        validSessionID,
				CurrentState:     "technical",
				InterviewStateID: "invalid-uuid",
			},
			setup: func() (*mockDatabase.MockRedisClient, *mockRepositories.MockEvaluationRubricsRepository, *mockRepositories.MockEvaluationScoresRepository, *mockRepositories.MockInterviewTurnsRepository, *mockUtils.MockGenerator) {
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockEvaluationRubricsRepo := new(mockRepositories.MockEvaluationRubricsRepository)
				mockEvaluationScoresRepo := new(mockRepositories.MockEvaluationScoresRepository)
				mockInterviewTurnsRepo := new(mockRepositories.MockInterviewTurnsRepository)
				mockGenerator := new(mockUtils.MockGenerator)

				return mockRedisClient, mockEvaluationRubricsRepo, mockEvaluationScoresRepo, mockInterviewTurnsRepo, mockGenerator
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Equal(t, app_error.ErrCodeGeneralInvalidUUID, gotErr.(*app_error.AppError).Code)
			},
		},
		{
			name:  "Error - RedisErrorNoRows",
			input: validReq,
			setup: func() (*mockDatabase.MockRedisClient, *mockRepositories.MockEvaluationRubricsRepository, *mockRepositories.MockEvaluationScoresRepository, *mockRepositories.MockInterviewTurnsRepository, *mockUtils.MockGenerator) {
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockEvaluationRubricsRepo := new(mockRepositories.MockEvaluationRubricsRepository)
				mockEvaluationScoresRepo := new(mockRepositories.MockEvaluationScoresRepository)
				mockInterviewTurnsRepo := new(mockRepositories.MockInterviewTurnsRepository)
				mockGenerator := new(mockUtils.MockGenerator)

				mockEvaluationScoresRepo.EXPECT().
					GetEvaluationSummaryJsonBySessionAndState(ctx, mock.MatchedBy(func(req *db.GetEvaluationSummaryJsonBySessionAndStateParams) bool {
						return req.SessionID == uuid.MustParse(validSessionID) &&
							req.CurrentState == validReq.CurrentState
					})).
					Return(nil, sql.ErrNoRows)

				return mockRedisClient, mockEvaluationRubricsRepo, mockEvaluationScoresRepo, mockInterviewTurnsRepo, mockGenerator
			},
			verify: func(t *testing.T, gotErr error) {
				assert.NoError(t, gotErr)
			},
		},
		{
			name:  "Error WithRedisError",
			input: validReq,
			setup: func() (*mockDatabase.MockRedisClient, *mockRepositories.MockEvaluationRubricsRepository, *mockRepositories.MockEvaluationScoresRepository, *mockRepositories.MockInterviewTurnsRepository, *mockUtils.MockGenerator) {
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockEvaluationRubricsRepo := new(mockRepositories.MockEvaluationRubricsRepository)
				mockEvaluationScoresRepo := new(mockRepositories.MockEvaluationScoresRepository)
				mockInterviewTurnsRepo := new(mockRepositories.MockInterviewTurnsRepository)
				mockGenerator := new(mockUtils.MockGenerator)

				mockEvaluationScoresRepo.EXPECT().
					GetEvaluationSummaryJsonBySessionAndState(ctx, mock.MatchedBy(func(req *db.GetEvaluationSummaryJsonBySessionAndStateParams) bool {
						return req.SessionID == uuid.MustParse(validSessionID) &&
							req.CurrentState == validReq.CurrentState
					})).
					Return(nil, errors.New("database error"))

				return mockRedisClient, mockEvaluationRubricsRepo, mockEvaluationScoresRepo, mockInterviewTurnsRepo, mockGenerator
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Equal(t, "database error", gotErr.Error())
			},
		},
		{
			name:  "Error WithGetEvaluationSummaryJsonBySessionAndStateFailed",
			input: validReq,
			setup: func() (*mockDatabase.MockRedisClient, *mockRepositories.MockEvaluationRubricsRepository, *mockRepositories.MockEvaluationScoresRepository, *mockRepositories.MockInterviewTurnsRepository, *mockUtils.MockGenerator) {
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockEvaluationRubricsRepo := new(mockRepositories.MockEvaluationRubricsRepository)
				mockEvaluationScoresRepo := new(mockRepositories.MockEvaluationScoresRepository)
				mockInterviewTurnsRepo := new(mockRepositories.MockInterviewTurnsRepository)
				mockGenerator := new(mockUtils.MockGenerator)

				// Mock GetEvaluationSummaryJsonBySessionAndState failure
				mockEvaluationScoresRepo.EXPECT().
					GetEvaluationSummaryJsonBySessionAndState(ctx, mock.MatchedBy(func(req *db.GetEvaluationSummaryJsonBySessionAndStateParams) bool {
						return req.SessionID == uuid.MustParse(validSessionID) &&
							req.CurrentState == validReq.CurrentState
					})).
					Return(nil, errors.New("database error"))

				return mockRedisClient, mockEvaluationRubricsRepo, mockEvaluationScoresRepo, mockInterviewTurnsRepo, mockGenerator
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Equal(t, "database error", gotErr.Error())
			},
		},
		{
			name:  "Error WithUnMarshallingEvaluationSummaryJsonFailed",
			input: validReq,
			setup: func() (*mockDatabase.MockRedisClient, *mockRepositories.MockEvaluationRubricsRepository, *mockRepositories.MockEvaluationScoresRepository, *mockRepositories.MockInterviewTurnsRepository, *mockUtils.MockGenerator) {
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockEvaluationRubricsRepo := new(mockRepositories.MockEvaluationRubricsRepository)
				mockEvaluationScoresRepo := new(mockRepositories.MockEvaluationScoresRepository)
				mockInterviewTurnsRepo := new(mockRepositories.MockInterviewTurnsRepository)
				mockGenerator := new(mockUtils.MockGenerator)

				// Mock GetEvaluationSummaryJsonBySessionAndState
				mockEvaluationScoresRepo.EXPECT().
					GetEvaluationSummaryJsonBySessionAndState(ctx, mock.MatchedBy(func(req *db.GetEvaluationSummaryJsonBySessionAndStateParams) bool {
						return req.SessionID == uuid.MustParse(validSessionID) &&
							req.CurrentState == validReq.CurrentState
					})).
					Return([]byte("invalid json"), nil)

				return mockRedisClient, mockEvaluationRubricsRepo, mockEvaluationScoresRepo, mockInterviewTurnsRepo, mockGenerator
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Equal(t, app_error.ErrCodeGeneralUnmarshalFailed, gotErr.(*app_error.AppError).Code)
			},
		},
		{
			name:  "Error WithCalculateEachCriteriaCommentBySessionAndStateFailed",
			input: validReq,
			setup: func() (*mockDatabase.MockRedisClient, *mockRepositories.MockEvaluationRubricsRepository, *mockRepositories.MockEvaluationScoresRepository, *mockRepositories.MockInterviewTurnsRepository, *mockUtils.MockGenerator) {
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockEvaluationRubricsRepo := new(mockRepositories.MockEvaluationRubricsRepository)
				mockEvaluationScoresRepo := new(mockRepositories.MockEvaluationScoresRepository)
				mockInterviewTurnsRepo := new(mockRepositories.MockInterviewTurnsRepository)
				mockGenerator := new(mockUtils.MockGenerator)

				// Mock GetEvaluationSummaryJsonBySessionAndState
				mockEvaluationScoresRepo.EXPECT().
					GetEvaluationSummaryJsonBySessionAndState(ctx, mock.MatchedBy(func(req *db.GetEvaluationSummaryJsonBySessionAndStateParams) bool {
						return req.SessionID == uuid.MustParse(validSessionID) &&
							req.CurrentState == validReq.CurrentState
					})).
					Return(jsonEvaluationSummaryResp, nil)

				// Mock CalculateEachCriteriaCommentBySessionAndState failure
				mockEvaluationScoresRepo.EXPECT().
					CalculateEachCriteriaCommentBySessionAndState(ctx, mock.MatchedBy(func(req *repositories.PreProcessedCriteriaReq) bool {
						return req.Criteria[0].CriteriaID == validCriteriaID1 &&
							req.Criteria[1].CriteriaID == validCriteriaID2
					})).
					Return(nil, errors.New("LLM service error"))

				return mockRedisClient, mockEvaluationRubricsRepo, mockEvaluationScoresRepo, mockInterviewTurnsRepo, mockGenerator
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Equal(t, "LLM service error", gotErr.Error())
			},
		},
		{
			name:  "Error WithInvalidCriteriaID",
			input: validReq,
			setup: func() (*mockDatabase.MockRedisClient, *mockRepositories.MockEvaluationRubricsRepository, *mockRepositories.MockEvaluationScoresRepository, *mockRepositories.MockInterviewTurnsRepository, *mockUtils.MockGenerator) {
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockEvaluationRubricsRepo := new(mockRepositories.MockEvaluationRubricsRepository)
				mockEvaluationScoresRepo := new(mockRepositories.MockEvaluationScoresRepository)
				mockInterviewTurnsRepo := new(mockRepositories.MockInterviewTurnsRepository)
				mockGenerator := new(mockUtils.MockGenerator)

				// Mock GetEvaluationSummaryJsonBySessionAndState
				mockEvaluationScoresRepo.EXPECT().
					GetEvaluationSummaryJsonBySessionAndState(ctx, mock.MatchedBy(func(req *db.GetEvaluationSummaryJsonBySessionAndStateParams) bool {
						return req.SessionID == uuid.MustParse(validSessionID) &&
							req.CurrentState == validReq.CurrentState
					})).
					Return(jsonEvaluationSummaryResp, nil)

				// Mock CalculateEachCriteriaCommentBySessionAndState
				mockEvaluationScoresRepo.EXPECT().
					CalculateEachCriteriaCommentBySessionAndState(ctx, mock.MatchedBy(func(req *repositories.PreProcessedCriteriaReq) bool {
						return req.Criteria[0].CriteriaID == validCriteriaID1 &&
							req.Criteria[1].CriteriaID == validCriteriaID2
					})).
					Return(inValidPostProcessedCriteriaResp, nil)

				// Mock generator calls
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(uuid.MustParse(validEvaluationID)).Times(1)
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(uuid.MustParse(validScoreID1)).Times(1)

				return mockRedisClient, mockEvaluationRubricsRepo, mockEvaluationScoresRepo, mockInterviewTurnsRepo, mockGenerator
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Equal(t, app_error.ErrCodeGeneralInvalidUUID, gotErr.(*app_error.AppError).Code)
			},
		},
		{
			name:  "Error WithCreatePhraseEvaluationAndCriteriaScoreWithIsScoredStateFailed",
			input: validReq,
			setup: func() (*mockDatabase.MockRedisClient, *mockRepositories.MockEvaluationRubricsRepository, *mockRepositories.MockEvaluationScoresRepository, *mockRepositories.MockInterviewTurnsRepository, *mockUtils.MockGenerator) {
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockEvaluationRubricsRepo := new(mockRepositories.MockEvaluationRubricsRepository)
				mockEvaluationScoresRepo := new(mockRepositories.MockEvaluationScoresRepository)
				mockInterviewTurnsRepo := new(mockRepositories.MockInterviewTurnsRepository)
				mockGenerator := new(mockUtils.MockGenerator)

				// Mock GetEvaluationSummaryJsonBySessionAndState
				mockEvaluationScoresRepo.EXPECT().
					GetEvaluationSummaryJsonBySessionAndState(ctx, mock.MatchedBy(func(req *db.GetEvaluationSummaryJsonBySessionAndStateParams) bool {
						return req.SessionID == uuid.MustParse(validSessionID) &&
							req.CurrentState == validReq.CurrentState
					})).
					Return(jsonEvaluationSummaryResp, nil)

				// Mock CalculateEachCriteriaCommentBySessionAndState
				mockEvaluationScoresRepo.EXPECT().
					CalculateEachCriteriaCommentBySessionAndState(ctx, mock.MatchedBy(func(req *repositories.PreProcessedCriteriaReq) bool {
						return req.Criteria[0].CriteriaID == validCriteriaID1 &&
							req.Criteria[1].CriteriaID == validCriteriaID2
					})).
					Return(validPostProcessedCriteriaResp, nil)

				// Mock generator calls
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(uuid.MustParse(validEvaluationID)).Times(1)
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(uuid.MustParse(validScoreID1)).Times(1)
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(uuid.MustParse(validScoreID2)).Times(1)

				// Mock CreatePhraseEvaluationAndCriteriaScoreWithIsScoredState failure
				mockEvaluationScoresRepo.EXPECT().
					CreatePhraseEvaluationAndCriteriaScoreWithIsScoredState(ctx, mock.MatchedBy(func(req *repositories.CreatePhraseEvaluationAndCriteriaScoreWithIsScoredStateReqTx) bool {
						return req.SessionID == uuid.MustParse(validSessionID) &&
							req.StateID == uuid.MustParse(validInterviewStateID) &&
							req.StateName == validReq.CurrentState &&
							req.OverallScore == validEvaluationSummaryResp.EvaluationAvgScore &&
							req.Criteria.CriteriaID[0] == uuid.MustParse(validCriteriaID1) &&
							req.Criteria.CriteriaID[1] == uuid.MustParse(validCriteriaID2) &&
							req.Criteria.CriteriaName[0] == "Problem Solving" &&
							req.Criteria.CriteriaName[1] == "Communication" &&
							req.Criteria.CriteriaAvgScore[0] == 8.0 &&
							req.Criteria.CriteriaAvgScore[1] == 7.0
					})).
					Return(errors.New("database error"))

				return mockRedisClient, mockEvaluationRubricsRepo, mockEvaluationScoresRepo, mockInterviewTurnsRepo, mockGenerator
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Equal(t, "database error", gotErr.Error())
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			mockRedisClient, mockEvaluationRubricsRepo, mockEvaluationScoresRepo, mockInterviewTurnsRepo, mockGenerator := tC.setup()

			defer func() {
				if mockRedisClient != nil {
					mockRedisClient.AssertExpectations(t)
				}
				if mockEvaluationRubricsRepo != nil {
					mockEvaluationRubricsRepo.AssertExpectations(t)
				}
				if mockEvaluationScoresRepo != nil {
					mockEvaluationScoresRepo.AssertExpectations(t)
				}
				if mockInterviewTurnsRepo != nil {
					mockInterviewTurnsRepo.AssertExpectations(t)
				}
				if mockGenerator != nil {
					mockGenerator.AssertExpectations(t)
				}
			}()

			svc := NewEvaluationService(
				lgr,
				mockRedisClient,
				mockEvaluationRubricsRepo,
				mockEvaluationScoresRepo,
				mockInterviewTurnsRepo,
				nil,
				nil,
				mockGenerator,
			)

			gotErr := svc.CalculateEvaluationInOldState(ctx, tC.input)

			tC.verify(t, gotErr)
		})
	}
}

func TestEvaluationService_GetPhraseEvaluationsWithCriteriaBySessionID(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()
	validSessionID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	validStateID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440001")
	validCriteriaID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440001")
	validCriteria := []entities.PhraseEvaluationCriteria{
		{
			CriteriaID:      validCriteriaID.String(),
			CriteriaName:    "test",
			CriteriaScore:   5,
			CriteriaComment: "test",
			CriteriaColor:   "#28A745",
		},
		{
			CriteriaID:      validCriteriaID.String(),
			CriteriaName:    "test",
			CriteriaScore:   5,
			CriteriaComment: "test",
			CriteriaColor:   "#28A745",
		},
	}

	dbResp := []db.GetPhraseEvaluationsWithCriteriaBySessionIDRow{
		{
			StateID:      validStateID,
			StateName:    "test",
			OverallScore: 1,
			Criteria: func() []byte {
				data, _ := json.Marshal(validCriteria)
				return data
			}(),
		},
	}

	invalidDbResp := []db.GetPhraseEvaluationsWithCriteriaBySessionIDRow{
		{
			StateID:      validStateID,
			StateName:    "test",
			OverallScore: 1,
			Criteria:     []byte("invalid json"),
		},
	}

	validResp := &entities.GetPhraseEvaluationsWithCriteriaResp{
		PhraseEvaluations: []entities.PhraseEvaluations{
			{
				StateID:      validStateID.String(),
				StateName:    "test",
				OverallScore: 1,
				MaxScore:     5,
				OverallColor: "#FD7E14",
				Criteria: []entities.PhraseEvaluationCriteria{
					{
						CriteriaID:      validCriteriaID.String(),
						CriteriaName:    "test",
						CriteriaScore:   5,
						CriteriaColor:   "#28A745",
						MaxScore:        5,
						CriteriaComment: "test",
					},
					{
						CriteriaID:      validCriteriaID.String(),
						CriteriaName:    "test",
						CriteriaScore:   5,
						CriteriaColor:   "#28A745",
						MaxScore:        5,
						CriteriaComment: "test",
					},
				},
			},
		},
	}

	testCases := []struct {
		name   string
		input  string
		setup  func() *mockRepositories.MockEvaluationScoresRepository
		verify func(t *testing.T, gotResp *entities.GetPhraseEvaluationsWithCriteriaResp, gotErr error)
	}{
		{
			name:  "Success",
			input: validSessionID.String(),
			setup: func() *mockRepositories.MockEvaluationScoresRepository {
				mockEvaluationScoresRepo := new(mockRepositories.MockEvaluationScoresRepository)

				mockEvaluationScoresRepo.EXPECT().
					GetPhraseEvaluationsWithCriteriaBySessionID(ctx, validSessionID).
					Return(dbResp, nil)

				return mockEvaluationScoresRepo
			},
			verify: func(t *testing.T, gotResp *entities.GetPhraseEvaluationsWithCriteriaResp, gotErr error) {
				assert.NoError(t, gotErr)
				assert.NotNil(t, gotResp)
				assert.Equal(t, validResp, gotResp)
			},
		},
		{
			name:  "Error WithInvalidSessionID",
			input: "invalid-uuid",
			setup: func() *mockRepositories.MockEvaluationScoresRepository {
				mockEvaluationScoresRepo := new(mockRepositories.MockEvaluationScoresRepository)

				return mockEvaluationScoresRepo
			},
			verify: func(t *testing.T, gotResp *entities.GetPhraseEvaluationsWithCriteriaResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, gotResp)
				assert.Equal(t, app_error.ErrCodeGeneralInvalidUUID, gotErr.(*app_error.AppError).Code)
			},
		},
		{
			name:  "Error WithGetPhraseEvaluationsWithCriteriaBySessionID Failed",
			input: validSessionID.String(),
			setup: func() *mockRepositories.MockEvaluationScoresRepository {
				mockEvaluationScoresRepo := new(mockRepositories.MockEvaluationScoresRepository)

				mockEvaluationScoresRepo.EXPECT().
					GetPhraseEvaluationsWithCriteriaBySessionID(ctx, validSessionID).
					Return(nil, errors.New("get phrase evaluations with criteria by session id error"))

				return mockEvaluationScoresRepo
			},
			verify: func(t *testing.T, gotResp *entities.GetPhraseEvaluationsWithCriteriaResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, gotResp)
				assert.Equal(t, "get phrase evaluations with criteria by session id error", gotErr.Error())
			},
		},
		{
			name:  "Error WithUnmarshalCriteriaFailed",
			input: validSessionID.String(),
			setup: func() *mockRepositories.MockEvaluationScoresRepository {
				mockEvaluationScoresRepo := new(mockRepositories.MockEvaluationScoresRepository)

				mockEvaluationScoresRepo.EXPECT().
					GetPhraseEvaluationsWithCriteriaBySessionID(ctx, validSessionID).
					Return(invalidDbResp, nil)

				return mockEvaluationScoresRepo
			},
			verify: func(t *testing.T, gotResp *entities.GetPhraseEvaluationsWithCriteriaResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, gotResp)
				assert.Equal(t, app_error.ErrCodeGeneralUnmarshalFailed, gotErr.(*app_error.AppError).Code)
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			mockEvaluationScoresRepo := tC.setup()

			defer func() {
				if mockEvaluationScoresRepo != nil {
					mockEvaluationScoresRepo.AssertExpectations(t)
				}
			}()

			svc := NewEvaluationService(
				lgr,
				nil,
				nil,
				mockEvaluationScoresRepo,
				nil,
				nil,
				nil,
				nil,
			)

			gotResp, gotErr := svc.GetPhraseEvaluationsWithCriteriaBySessionID(ctx, tC.input)

			tC.verify(t, gotResp, gotErr)
		})
	}
}

func TestEvaluationService_FinalizeSessionPhraseEvaluation(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()

	// Test data
	validSessionID := "550e8400-e29b-41d4-a716-446655440000"
	validStateID1 := "550e8400-e29b-41d4-a716-446655440001"
	validStateID2 := "550e8400-e29b-41d4-a716-446655440002"
	validCriteriaID := "550e8400-e29b-41d4-a716-446655440003"

	validStates := []db.GetUnprocessedInterviewStatesBySessionIDRow{
		{
			ID:         uuid.MustParse(validStateID1),
			PhraseType: "technical",
		},
		{
			ID:         uuid.MustParse(validStateID2),
			PhraseType: "behavioral",
		},
	}

	validEvaluationSummaryResp := &entities.GetEvaluationSummaryJsonBySessionAndStateResp{
		Criteria: []entities.EvaluationCriteria{
			{
				CriteriaID:   validCriteriaID,
				CriteriaName: "Problem Solving",
				CriteriaScore: []entities.CriteriaScoreOld{
					{Score: 8, Comment: "Good approach"},
					{Score: 9, Comment: "Excellent solution"},
				},
			},
		},
		EvaluationAvgScore: 8.5,
	}

	validPostProcessedCriteriaResp := &repositories.PostProcessedCriteriaResp{
		Criteria: []repositories.PostProcessedCriteria{
			{
				CriteriaID:       validCriteriaID,
				CriteriaName:     "Problem Solving",
				CriteriaAvgScore: 8.5,
				CriteriaComment:  "Good problem solving skills",
			},
		},
	}

	jsonEvaluationSummaryResp, err := json.Marshal(validEvaluationSummaryResp)
	if err != nil {
		t.Fatalf("Failed to marshal evaluation summary response: %v", err)
	}

	testCases := []struct {
		name   string
		input  string
		setup  func() (*mockRepositories.MockInterviewSessionRepository, *mockRepositories.MockInterviewStateRepository, *mockRepositories.MockEvaluationScoresRepository, *mockUtils.MockGenerator)
		verify func(t *testing.T, gotErr error)
	}{
		{
			name:  "Success",
			input: validSessionID,
			setup: func() (*mockRepositories.MockInterviewSessionRepository, *mockRepositories.MockInterviewStateRepository, *mockRepositories.MockEvaluationScoresRepository, *mockUtils.MockGenerator) {
				mockInterviewSessionsRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockInterviewStatesRepo := new(mockRepositories.MockInterviewStateRepository)
				mockEvaluationScoresRepo := new(mockRepositories.MockEvaluationScoresRepository)
				mockGenerator := new(mockUtils.MockGenerator)

				// Mock UpdateFinalizeStatusInterviewSessionByID (first call)
				mockInterviewSessionsRepo.EXPECT().
					UpdateFinalizeStatusInterviewSessionByID(ctx, mock.MatchedBy(func(req *db.UpdateFinalizeStatusInterviewSessionByIDParams) bool {
						return req.ID == uuid.MustParse(validSessionID) &&
							req.FinalizeStatus.FinalizeStatusEnum == db.FinalizeStatusEnumFinalizing
					})).
					Return(nil)

				// Mock GetUnprocessedInterviewStatesBySessionID
				mockInterviewStatesRepo.EXPECT().
					GetUnprocessedInterviewStatesBySessionID(ctx, uuid.MustParse(validSessionID)).
					Return(validStates, nil)

				// Mock CalculateEvaluationInOldState for each state
				for i := range validStates {
					// Mock GetEvaluationSummaryJsonBySessionAndState
					mockEvaluationScoresRepo.EXPECT().
						GetEvaluationSummaryJsonBySessionAndState(ctx, mock.AnythingOfType("*db.GetEvaluationSummaryJsonBySessionAndStateParams")).
						Return(jsonEvaluationSummaryResp, nil)

					// Mock CalculateEachCriteriaCommentBySessionAndState
					mockEvaluationScoresRepo.EXPECT().
						CalculateEachCriteriaCommentBySessionAndState(ctx, mock.AnythingOfType("*repositories.PreProcessedCriteriaReq")).
						Return(validPostProcessedCriteriaResp, nil)

					mockGenerator.EXPECT().GenerateUUID(ctx).Return(uuid.MustParse("550e8400-e29b-41d4-a716-446655440008")).Times(1)
					mockGenerator.EXPECT().GenerateUUID(ctx).Return(uuid.MustParse("550e8400-e29b-41d4-a716-446655440009")).Times(1)

					mockEvaluationScoresRepo.EXPECT().
						CreatePhraseEvaluationAndCriteriaScoreWithIsScoredState(ctx, mock.MatchedBy(func(req *repositories.CreatePhraseEvaluationAndCriteriaScoreWithIsScoredStateReqTx) bool {
							return req.SessionID == uuid.MustParse(validSessionID) &&
								req.StateID == validStates[i].ID &&
								req.StateName == validStates[i].PhraseType
						})).
						Return(nil)
				}

				// Mock UpdateFinalizeStatusInterviewSessionByID (second call)
				mockInterviewSessionsRepo.EXPECT().
					UpdateFinalizeStatusInterviewSessionByID(ctx, mock.MatchedBy(func(req *db.UpdateFinalizeStatusInterviewSessionByIDParams) bool {
						return req.ID == uuid.MustParse(validSessionID) &&
							req.FinalizeStatus.FinalizeStatusEnum == db.FinalizeStatusEnumFinalized
					})).
					Return(nil)

				return mockInterviewSessionsRepo, mockInterviewStatesRepo, mockEvaluationScoresRepo, mockGenerator
			},
			verify: func(t *testing.T, gotErr error) {
				assert.NoError(t, gotErr)
			},
		},
		{
			name:  "Error WithInvalidSessionID",
			input: "invalid-uuid",
			setup: func() (*mockRepositories.MockInterviewSessionRepository, *mockRepositories.MockInterviewStateRepository, *mockRepositories.MockEvaluationScoresRepository, *mockUtils.MockGenerator) {
				mockInterviewSessionsRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockInterviewStatesRepo := new(mockRepositories.MockInterviewStateRepository)
				mockEvaluationScoresRepo := new(mockRepositories.MockEvaluationScoresRepository)
				mockGenerator := new(mockUtils.MockGenerator)

				return mockInterviewSessionsRepo, mockInterviewStatesRepo, mockEvaluationScoresRepo, mockGenerator
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Equal(t, app_error.ErrCodeGeneralInvalidUUID, gotErr.(*app_error.AppError).Code)
			},
		},
		{
			name:  "Error WithUpdateFinalizeStatusInterviewSessionByIDFailed(First Call)",
			input: validSessionID,
			setup: func() (*mockRepositories.MockInterviewSessionRepository, *mockRepositories.MockInterviewStateRepository, *mockRepositories.MockEvaluationScoresRepository, *mockUtils.MockGenerator) {
				mockInterviewSessionsRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockInterviewStatesRepo := new(mockRepositories.MockInterviewStateRepository)
				mockEvaluationScoresRepo := new(mockRepositories.MockEvaluationScoresRepository)
				mockGenerator := new(mockUtils.MockGenerator)

				mockInterviewSessionsRepo.EXPECT().
					UpdateFinalizeStatusInterviewSessionByID(ctx, mock.MatchedBy(func(req *db.UpdateFinalizeStatusInterviewSessionByIDParams) bool {
						return req.ID == uuid.MustParse(validSessionID) &&
							req.FinalizeStatus.FinalizeStatusEnum == db.FinalizeStatusEnumFinalizing
					})).
					Return(errors.New("database error"))

				mockInterviewSessionsRepo.EXPECT().
					UpdateFinalizeStatusInterviewSessionByID(ctx, mock.MatchedBy(func(req *db.UpdateFinalizeStatusInterviewSessionByIDParams) bool {
						return req.ID == uuid.MustParse(validSessionID) &&
							req.FinalizeStatus.FinalizeStatusEnum == db.FinalizeStatusEnumFailed
					})).
					Return(nil)

				return mockInterviewSessionsRepo, mockInterviewStatesRepo, mockEvaluationScoresRepo, mockGenerator
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Equal(t, "database error", gotErr.Error())
			},
		},
		{
			name:  "Error WithGetUnprocessedInterviewStatesBySessionIDFailed",
			input: validSessionID,
			setup: func() (*mockRepositories.MockInterviewSessionRepository, *mockRepositories.MockInterviewStateRepository, *mockRepositories.MockEvaluationScoresRepository, *mockUtils.MockGenerator) {
				mockInterviewSessionsRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockInterviewStatesRepo := new(mockRepositories.MockInterviewStateRepository)
				mockEvaluationScoresRepo := new(mockRepositories.MockEvaluationScoresRepository)
				mockGenerator := new(mockUtils.MockGenerator)

				// Mock UpdateFinalizeStatusInterviewSessionByID (first call)
				mockInterviewSessionsRepo.EXPECT().
					UpdateFinalizeStatusInterviewSessionByID(ctx, mock.MatchedBy(func(req *db.UpdateFinalizeStatusInterviewSessionByIDParams) bool {
						return req.ID == uuid.MustParse(validSessionID) &&
							req.FinalizeStatus.FinalizeStatusEnum == db.FinalizeStatusEnumFinalizing
					})).
					Return(nil)

				mockInterviewStatesRepo.EXPECT().
					GetUnprocessedInterviewStatesBySessionID(ctx, uuid.MustParse(validSessionID)).
					Return(nil, errors.New("database error"))

				mockInterviewSessionsRepo.EXPECT().
					UpdateFinalizeStatusInterviewSessionByID(ctx, mock.MatchedBy(func(req *db.UpdateFinalizeStatusInterviewSessionByIDParams) bool {
						return req.ID == uuid.MustParse(validSessionID) &&
							req.FinalizeStatus.FinalizeStatusEnum == db.FinalizeStatusEnumFailed
					})).
					Return(nil)

				return mockInterviewSessionsRepo, mockInterviewStatesRepo, mockEvaluationScoresRepo, mockGenerator
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Equal(t, "database error", gotErr.Error())
			},
		},
		{
			name:  "Error WithCalculateEvaluationInOldStateFailed",
			input: validSessionID,
			setup: func() (*mockRepositories.MockInterviewSessionRepository, *mockRepositories.MockInterviewStateRepository, *mockRepositories.MockEvaluationScoresRepository, *mockUtils.MockGenerator) {
				mockInterviewSessionsRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockInterviewStatesRepo := new(mockRepositories.MockInterviewStateRepository)
				mockEvaluationScoresRepo := new(mockRepositories.MockEvaluationScoresRepository)
				mockGenerator := new(mockUtils.MockGenerator)

				// Mock UpdateFinalizeStatusInterviewSessionByID (first call)
				mockInterviewSessionsRepo.EXPECT().
					UpdateFinalizeStatusInterviewSessionByID(ctx, mock.MatchedBy(func(req *db.UpdateFinalizeStatusInterviewSessionByIDParams) bool {
						return req.ID == uuid.MustParse(validSessionID) &&
							req.FinalizeStatus.FinalizeStatusEnum == db.FinalizeStatusEnumFinalizing
					})).
					Return(nil)

				// Mock GetUnprocessedInterviewStatesBySessionID
				mockInterviewStatesRepo.EXPECT().
					GetUnprocessedInterviewStatesBySessionID(ctx, uuid.MustParse(validSessionID)).
					Return(validStates, nil)

				// Mock CalculateEvaluationInOldState failure for first state
				mockEvaluationScoresRepo.EXPECT().
					GetEvaluationSummaryJsonBySessionAndState(ctx, mock.AnythingOfType("*db.GetEvaluationSummaryJsonBySessionAndStateParams")).
					Return(nil, errors.New("database error"))

				// Mock FinalizeSessionFailed call
				mockInterviewSessionsRepo.EXPECT().
					UpdateFinalizeStatusInterviewSessionByID(ctx, mock.MatchedBy(func(req *db.UpdateFinalizeStatusInterviewSessionByIDParams) bool {
						return req.ID == uuid.MustParse(validSessionID) &&
							req.FinalizeStatus.FinalizeStatusEnum == db.FinalizeStatusEnumFailed
					})).
					Return(nil)

				return mockInterviewSessionsRepo, mockInterviewStatesRepo, mockEvaluationScoresRepo, mockGenerator
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Equal(t, "database error", gotErr.Error())
			},
		},
		{
			name:  "Success - No Unprocessed States",
			input: validSessionID,
			setup: func() (*mockRepositories.MockInterviewSessionRepository, *mockRepositories.MockInterviewStateRepository, *mockRepositories.MockEvaluationScoresRepository, *mockUtils.MockGenerator) {
				mockInterviewSessionsRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockInterviewStatesRepo := new(mockRepositories.MockInterviewStateRepository)
				mockEvaluationScoresRepo := new(mockRepositories.MockEvaluationScoresRepository)
				mockGenerator := new(mockUtils.MockGenerator)

				// Mock UpdateFinalizeStatusInterviewSessionByID (first call)
				mockInterviewSessionsRepo.EXPECT().
					UpdateFinalizeStatusInterviewSessionByID(ctx, mock.MatchedBy(func(req *db.UpdateFinalizeStatusInterviewSessionByIDParams) bool {
						return req.ID == uuid.MustParse(validSessionID) &&
							req.FinalizeStatus.FinalizeStatusEnum == db.FinalizeStatusEnumFinalizing
					})).
					Return(nil)

				// Mock GetUnprocessedInterviewStatesBySessionID - return empty
				mockInterviewStatesRepo.EXPECT().
					GetUnprocessedInterviewStatesBySessionID(ctx, uuid.MustParse(validSessionID)).
					Return([]db.GetUnprocessedInterviewStatesBySessionIDRow{}, nil)

				// Mock UpdateFinalizeStatusInterviewSessionByID (second call)
				mockInterviewSessionsRepo.EXPECT().
					UpdateFinalizeStatusInterviewSessionByID(ctx, mock.MatchedBy(func(req *db.UpdateFinalizeStatusInterviewSessionByIDParams) bool {
						return req.ID == uuid.MustParse(validSessionID) &&
							req.FinalizeStatus.FinalizeStatusEnum == db.FinalizeStatusEnumFinalized
					})).
					Return(nil)

				return mockInterviewSessionsRepo, mockInterviewStatesRepo, mockEvaluationScoresRepo, mockGenerator
			},
			verify: func(t *testing.T, gotErr error) {
				assert.NoError(t, gotErr)
			},
		},
		{
			name:  "Error WithUpdateFinalizedStatusFailed",
			input: validSessionID,
			setup: func() (*mockRepositories.MockInterviewSessionRepository, *mockRepositories.MockInterviewStateRepository, *mockRepositories.MockEvaluationScoresRepository, *mockUtils.MockGenerator) {
				mockInterviewSessionsRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockInterviewStatesRepo := new(mockRepositories.MockInterviewStateRepository)
				mockEvaluationScoresRepo := new(mockRepositories.MockEvaluationScoresRepository)
				mockGenerator := new(mockUtils.MockGenerator)

				// Mock UpdateFinalizeStatusInterviewSessionByID (first call)
				mockInterviewSessionsRepo.EXPECT().
					UpdateFinalizeStatusInterviewSessionByID(ctx, mock.MatchedBy(func(req *db.UpdateFinalizeStatusInterviewSessionByIDParams) bool {
						return req.ID == uuid.MustParse(validSessionID) &&
							req.FinalizeStatus.FinalizeStatusEnum == db.FinalizeStatusEnumFinalizing
					})).
					Return(nil)

				// Mock GetUnprocessedInterviewStatesBySessionID
				mockInterviewStatesRepo.EXPECT().
					GetUnprocessedInterviewStatesBySessionID(ctx, uuid.MustParse(validSessionID)).
					Return(validStates, nil)

				// Mock CalculateEvaluationInOldState for each state
				for range validStates {
					// Mock GetEvaluationSummaryJsonBySessionAndState
					mockEvaluationScoresRepo.EXPECT().
						GetEvaluationSummaryJsonBySessionAndState(ctx, mock.AnythingOfType("*db.GetEvaluationSummaryJsonBySessionAndStateParams")).
						Return(jsonEvaluationSummaryResp, nil)

					// Mock CalculateEachCriteriaCommentBySessionAndState
					mockEvaluationScoresRepo.EXPECT().
						CalculateEachCriteriaCommentBySessionAndState(ctx, mock.AnythingOfType("*repositories.PreProcessedCriteriaReq")).
						Return(validPostProcessedCriteriaResp, nil)

					// Mock generator calls for each state
					mockGenerator.EXPECT().GenerateUUID(ctx).Return(uuid.MustParse("550e8400-e29b-41d4-a716-446655440008")).Times(1)
					mockGenerator.EXPECT().GenerateUUID(ctx).Return(uuid.MustParse("550e8400-e29b-41d4-a716-446655440009")).Times(1)

					// Mock CreatePhraseEvaluationAndCriteriaScoreWithIsScoredState
					mockEvaluationScoresRepo.EXPECT().
						CreatePhraseEvaluationAndCriteriaScoreWithIsScoredState(ctx, mock.AnythingOfType("*repositories.CreatePhraseEvaluationAndCriteriaScoreWithIsScoredStateReqTx")).
						Return(nil)
				}

				mockInterviewSessionsRepo.EXPECT().
					UpdateFinalizeStatusInterviewSessionByID(ctx, mock.MatchedBy(func(req *db.UpdateFinalizeStatusInterviewSessionByIDParams) bool {
						return req.ID == uuid.MustParse(validSessionID) &&
							req.FinalizeStatus.FinalizeStatusEnum == db.FinalizeStatusEnumFinalized
					})).
					Return(errors.New("database error"))

				mockInterviewSessionsRepo.EXPECT().
					UpdateFinalizeStatusInterviewSessionByID(ctx, mock.MatchedBy(func(req *db.UpdateFinalizeStatusInterviewSessionByIDParams) bool {
						return req.ID == uuid.MustParse(validSessionID) &&
							req.FinalizeStatus.FinalizeStatusEnum == db.FinalizeStatusEnumFailed
					})).
					Return(nil)

				return mockInterviewSessionsRepo, mockInterviewStatesRepo, mockEvaluationScoresRepo, mockGenerator
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Equal(t, "database error", gotErr.Error())
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			mockInterviewSessionsRepo, mockInterviewStatesRepo, mockEvaluationScoresRepo, mockGenerator := tC.setup()

			defer func() {
				if mockInterviewSessionsRepo != nil {
					mockInterviewSessionsRepo.AssertExpectations(t)
				}
				if mockInterviewStatesRepo != nil {
					mockInterviewStatesRepo.AssertExpectations(t)
				}
				if mockEvaluationScoresRepo != nil {
					mockEvaluationScoresRepo.AssertExpectations(t)
				}
				if mockGenerator != nil {
					mockGenerator.AssertExpectations(t)
				}
			}()

			svc := NewEvaluationService(
				lgr,
				nil,
				nil,
				mockEvaluationScoresRepo,
				nil,
				mockInterviewSessionsRepo,
				mockInterviewStatesRepo,
				mockGenerator,
			)

			gotErr := svc.FinalizeSessionPhraseEvaluation(ctx, tC.input)

			tC.verify(t, gotErr)
		})
	}
}

func TestEvaluationService_FinalizeSessionFailed(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()

	// Test data
	validSessionID := "550e8400-e29b-41d4-a716-446655440000"

	testCases := []struct {
		name   string
		input  string
		setup  func() *mockRepositories.MockInterviewSessionRepository
		verify func(t *testing.T)
	}{
		{
			name:  "Success - Complete Flow",
			input: validSessionID,
			setup: func() *mockRepositories.MockInterviewSessionRepository {
				mockInterviewSessionsRepo := new(mockRepositories.MockInterviewSessionRepository)

				// Mock UpdateFinalizeStatusInterviewSessionByID
				mockInterviewSessionsRepo.EXPECT().
					UpdateFinalizeStatusInterviewSessionByID(ctx, mock.AnythingOfType("*db.UpdateFinalizeStatusInterviewSessionByIDParams")).
					Return(nil)

				return mockInterviewSessionsRepo
			},
			verify: func(t *testing.T) {
				// This method doesn't return an error, so we just verify it completes
			},
		},
		{
			name:  "Error - Invalid Session ID",
			input: "invalid-uuid",
			setup: func() *mockRepositories.MockInterviewSessionRepository {
				mockInterviewSessionsRepo := new(mockRepositories.MockInterviewSessionRepository)

				return mockInterviewSessionsRepo
			},
			verify: func(t *testing.T) {
				// This method doesn't return an error, so we just verify it completes
			},
		},
		{
			name:  "Error - UpdateFinalizeStatusInterviewSessionByID Failed",
			input: validSessionID,
			setup: func() *mockRepositories.MockInterviewSessionRepository {
				mockInterviewSessionsRepo := new(mockRepositories.MockInterviewSessionRepository)

				// Mock UpdateFinalizeStatusInterviewSessionByID failure
				mockInterviewSessionsRepo.EXPECT().
					UpdateFinalizeStatusInterviewSessionByID(ctx, mock.AnythingOfType("*db.UpdateFinalizeStatusInterviewSessionByIDParams")).
					Return(errors.New("database error"))

				return mockInterviewSessionsRepo
			},
			verify: func(t *testing.T) {
				// This method doesn't return an error, so we just verify it completes
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			mockInterviewSessionsRepo := tC.setup()

			defer func() {
				if mockInterviewSessionsRepo != nil {
					mockInterviewSessionsRepo.AssertExpectations(t)
				}
			}()

			svc := NewEvaluationService(
				lgr,
				nil,
				nil,
				nil,
				nil,
				mockInterviewSessionsRepo,
				nil,
				nil,
			)

			svc.FinalizeSessionFailed(ctx, tC.input)

			tC.verify(t)
		})
	}
}
