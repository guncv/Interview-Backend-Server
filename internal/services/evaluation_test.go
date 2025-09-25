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

				// Note: FlagIsScoreEvaluated and Redis Set calls are unreachable due to early return in service

				return mockRedisClient, mockEvaluationRubricsRepo, mockEvaluationScoresRepo, mockInterviewTurnsRepo, mockGenerator
			},
			verify: func(t *testing.T, gotErr error) {
				assert.NoError(t, gotErr)
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
			name:  "Error - CreateEvaluationWithCriteriaScoreAndImproveSentenceTx Failed",
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

				// Mock CreateEvaluationWithCriteriaScoreAndImproveSentenceTx failure (retry scenario)
				mockEvaluationScoresRepo.EXPECT().
					CreateEvaluationWithCriteriaScoreAndImproveSentenceTx(ctx, mock.AnythingOfType("*repositories.CreateEvaluationAndScoreTxReq")).
					Return(errors.New("database error")).
					Times(constants.MaxRetryDbEvaluationTx)

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

	// Test data
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

				// Mock Redis check for scored state
				redisKey := fmt.Sprintf("%s%s:%s", constants.RedisPrefixInterviewIsScoreSessionState, validReq.SessionID, validReq.CurrentState)
				mockRedisClient.EXPECT().
					Get(ctx, redisKey).
					Return("true", nil)

				// Mock GetEvaluationSummaryJsonBySessionAndState
				mockEvaluationScoresRepo.EXPECT().
					GetEvaluationSummaryJsonBySessionAndState(ctx, mock.AnythingOfType("*db.GetEvaluationSummaryJsonBySessionAndStateParams")).
					Return(jsonEvaluationSummaryResp, nil)

				// Mock CalculateEachCriteriaCommentBySessionAndState
				mockEvaluationScoresRepo.EXPECT().
					CalculateEachCriteriaCommentBySessionAndState(ctx, mock.AnythingOfType("*repositories.PreProcessedCriteriaReq")).
					Return(validPostProcessedCriteriaResp, nil)

				// Mock generator calls
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(uuid.MustParse(validEvaluationID)).Times(1)
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(uuid.MustParse(validScoreID1)).Times(1)
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(uuid.MustParse(validScoreID2)).Times(1)

				// Mock CreatePhraseEvaluationAndCriteriaScoreWithIsScoredState
				mockEvaluationScoresRepo.EXPECT().
					CreatePhraseEvaluationAndCriteriaScoreWithIsScoredState(ctx, mock.AnythingOfType("*repositories.CreatePhraseEvaluationAndCriteriaScoreWithIsScoredStateReqTx")).
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

				// Mock Redis miss
				redisKey := fmt.Sprintf("%s%s:%s", constants.RedisPrefixInterviewIsScoreSessionState, validReq.SessionID, validReq.CurrentState)
				mockRedisClient.EXPECT().
					Get(ctx, redisKey).
					Return("", errors.New("redis miss"))

				// Mock DB check for scored state
				mockEvaluationScoresRepo.EXPECT().
					IsLastUserStateTurnScored(ctx, mock.AnythingOfType("*db.IsLastUserStateTurnScoredParams")).
					Return(true, nil)

				// Mock GetEvaluationSummaryJsonBySessionAndState
				mockEvaluationScoresRepo.EXPECT().
					GetEvaluationSummaryJsonBySessionAndState(ctx, mock.AnythingOfType("*db.GetEvaluationSummaryJsonBySessionAndStateParams")).
					Return(jsonEvaluationSummaryResp, nil)

				// Mock CalculateEachCriteriaCommentBySessionAndState
				mockEvaluationScoresRepo.EXPECT().
					CalculateEachCriteriaCommentBySessionAndState(ctx, mock.AnythingOfType("*repositories.PreProcessedCriteriaReq")).
					Return(validPostProcessedCriteriaResp, nil)

				// Mock generator calls
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(uuid.MustParse(validEvaluationID)).Times(1)
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(uuid.MustParse(validScoreID1)).Times(1)
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(uuid.MustParse(validScoreID2)).Times(1)

				// Mock CreatePhraseEvaluationAndCriteriaScoreWithIsScoredState
				mockEvaluationScoresRepo.EXPECT().
					CreatePhraseEvaluationAndCriteriaScoreWithIsScoredState(ctx, mock.AnythingOfType("*repositories.CreatePhraseEvaluationAndCriteriaScoreWithIsScoredStateReqTx")).
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
			name:  "Error - Redis Parse Error",
			input: validReq,
			setup: func() (*mockDatabase.MockRedisClient, *mockRepositories.MockEvaluationRubricsRepository, *mockRepositories.MockEvaluationScoresRepository, *mockRepositories.MockInterviewTurnsRepository, *mockUtils.MockGenerator) {
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockEvaluationRubricsRepo := new(mockRepositories.MockEvaluationRubricsRepository)
				mockEvaluationScoresRepo := new(mockRepositories.MockEvaluationScoresRepository)
				mockInterviewTurnsRepo := new(mockRepositories.MockInterviewTurnsRepository)
				mockGenerator := new(mockUtils.MockGenerator)

				// Mock Redis with invalid boolean value
				redisKey := fmt.Sprintf("%s%s:%s", constants.RedisPrefixInterviewIsScoreSessionState, validReq.SessionID, validReq.CurrentState)
				mockRedisClient.EXPECT().
					Get(ctx, redisKey).
					Return("invalid-bool", nil)

				return mockRedisClient, mockEvaluationRubricsRepo, mockEvaluationScoresRepo, mockInterviewTurnsRepo, mockGenerator
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
			},
		},
		{
			name:  "Error - IsLastUserStateTurnScored Failed",
			input: validReq,
			setup: func() (*mockDatabase.MockRedisClient, *mockRepositories.MockEvaluationRubricsRepository, *mockRepositories.MockEvaluationScoresRepository, *mockRepositories.MockInterviewTurnsRepository, *mockUtils.MockGenerator) {
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockEvaluationRubricsRepo := new(mockRepositories.MockEvaluationRubricsRepository)
				mockEvaluationScoresRepo := new(mockRepositories.MockEvaluationScoresRepository)
				mockInterviewTurnsRepo := new(mockRepositories.MockInterviewTurnsRepository)
				mockGenerator := new(mockUtils.MockGenerator)

				// Mock Redis miss
				redisKey := fmt.Sprintf("%s%s:%s", constants.RedisPrefixInterviewIsScoreSessionState, validReq.SessionID, validReq.CurrentState)
				mockRedisClient.EXPECT().
					Get(ctx, redisKey).
					Return("", errors.New("redis miss"))

				// Mock DB check failure
				mockEvaluationScoresRepo.EXPECT().
					IsLastUserStateTurnScored(ctx, mock.AnythingOfType("*db.IsLastUserStateTurnScoredParams")).
					Return(false, errors.New("database error"))

				return mockRedisClient, mockEvaluationRubricsRepo, mockEvaluationScoresRepo, mockInterviewTurnsRepo, mockGenerator
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Equal(t, "database error", gotErr.Error())
			},
		},
		{
			name:  "Error - GetEvaluationSummaryJsonBySessionAndState Failed",
			input: validReq,
			setup: func() (*mockDatabase.MockRedisClient, *mockRepositories.MockEvaluationRubricsRepository, *mockRepositories.MockEvaluationScoresRepository, *mockRepositories.MockInterviewTurnsRepository, *mockUtils.MockGenerator) {
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockEvaluationRubricsRepo := new(mockRepositories.MockEvaluationRubricsRepository)
				mockEvaluationScoresRepo := new(mockRepositories.MockEvaluationScoresRepository)
				mockInterviewTurnsRepo := new(mockRepositories.MockInterviewTurnsRepository)
				mockGenerator := new(mockUtils.MockGenerator)

				// Mock Redis check for scored state
				redisKey := fmt.Sprintf("%s%s:%s", constants.RedisPrefixInterviewIsScoreSessionState, validReq.SessionID, validReq.CurrentState)
				mockRedisClient.EXPECT().
					Get(ctx, redisKey).
					Return("true", nil)

				// Mock GetEvaluationSummaryJsonBySessionAndState failure
				mockEvaluationScoresRepo.EXPECT().
					GetEvaluationSummaryJsonBySessionAndState(ctx, mock.AnythingOfType("*db.GetEvaluationSummaryJsonBySessionAndStateParams")).
					Return(nil, errors.New("database error"))

				return mockRedisClient, mockEvaluationRubricsRepo, mockEvaluationScoresRepo, mockInterviewTurnsRepo, mockGenerator
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Equal(t, "database error", gotErr.Error())
			},
		},
		{
			name:  "Error - CalculateEachCriteriaCommentBySessionAndState Failed",
			input: validReq,
			setup: func() (*mockDatabase.MockRedisClient, *mockRepositories.MockEvaluationRubricsRepository, *mockRepositories.MockEvaluationScoresRepository, *mockRepositories.MockInterviewTurnsRepository, *mockUtils.MockGenerator) {
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockEvaluationRubricsRepo := new(mockRepositories.MockEvaluationRubricsRepository)
				mockEvaluationScoresRepo := new(mockRepositories.MockEvaluationScoresRepository)
				mockInterviewTurnsRepo := new(mockRepositories.MockInterviewTurnsRepository)
				mockGenerator := new(mockUtils.MockGenerator)

				// Mock Redis check for scored state
				redisKey := fmt.Sprintf("%s%s:%s", constants.RedisPrefixInterviewIsScoreSessionState, validReq.SessionID, validReq.CurrentState)
				mockRedisClient.EXPECT().
					Get(ctx, redisKey).
					Return("true", nil)

				// Mock GetEvaluationSummaryJsonBySessionAndState
				mockEvaluationScoresRepo.EXPECT().
					GetEvaluationSummaryJsonBySessionAndState(ctx, mock.AnythingOfType("*db.GetEvaluationSummaryJsonBySessionAndStateParams")).
					Return(jsonEvaluationSummaryResp, nil)

				// Mock CalculateEachCriteriaCommentBySessionAndState failure
				mockEvaluationScoresRepo.EXPECT().
					CalculateEachCriteriaCommentBySessionAndState(ctx, mock.AnythingOfType("*repositories.PreProcessedCriteriaReq")).
					Return(nil, errors.New("LLM service error"))

				return mockRedisClient, mockEvaluationRubricsRepo, mockEvaluationScoresRepo, mockInterviewTurnsRepo, mockGenerator
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Equal(t, "LLM service error", gotErr.Error())
			},
		},
		{
			name:  "Error - CreatePhraseEvaluationAndCriteriaScoreWithIsScoredState Failed",
			input: validReq,
			setup: func() (*mockDatabase.MockRedisClient, *mockRepositories.MockEvaluationRubricsRepository, *mockRepositories.MockEvaluationScoresRepository, *mockRepositories.MockInterviewTurnsRepository, *mockUtils.MockGenerator) {
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockEvaluationRubricsRepo := new(mockRepositories.MockEvaluationRubricsRepository)
				mockEvaluationScoresRepo := new(mockRepositories.MockEvaluationScoresRepository)
				mockInterviewTurnsRepo := new(mockRepositories.MockInterviewTurnsRepository)
				mockGenerator := new(mockUtils.MockGenerator)

				// Mock Redis check for scored state
				redisKey := fmt.Sprintf("%s%s:%s", constants.RedisPrefixInterviewIsScoreSessionState, validReq.SessionID, validReq.CurrentState)
				mockRedisClient.EXPECT().
					Get(ctx, redisKey).
					Return("true", nil)

				// Mock GetEvaluationSummaryJsonBySessionAndState
				mockEvaluationScoresRepo.EXPECT().
					GetEvaluationSummaryJsonBySessionAndState(ctx, mock.AnythingOfType("*db.GetEvaluationSummaryJsonBySessionAndStateParams")).
					Return(jsonEvaluationSummaryResp, nil)

				// Mock CalculateEachCriteriaCommentBySessionAndState
				mockEvaluationScoresRepo.EXPECT().
					CalculateEachCriteriaCommentBySessionAndState(ctx, mock.AnythingOfType("*repositories.PreProcessedCriteriaReq")).
					Return(validPostProcessedCriteriaResp, nil)

				// Mock generator calls
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(uuid.MustParse(validEvaluationID)).Times(1)
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(uuid.MustParse(validScoreID1)).Times(1)
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(uuid.MustParse(validScoreID2)).Times(1)

				// Mock CreatePhraseEvaluationAndCriteriaScoreWithIsScoredState failure
				mockEvaluationScoresRepo.EXPECT().
					CreatePhraseEvaluationAndCriteriaScoreWithIsScoredState(ctx, mock.AnythingOfType("*repositories.CreatePhraseEvaluationAndCriteriaScoreWithIsScoredStateReqTx")).
					Return(errors.New("database error"))

				return mockRedisClient, mockEvaluationRubricsRepo, mockEvaluationScoresRepo, mockInterviewTurnsRepo, mockGenerator
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Equal(t, "database error", gotErr.Error())
			},
		},
		{
			name:  "Success - Retry Logic - Turn Not Scored Yet",
			input: validReq,
			setup: func() (*mockDatabase.MockRedisClient, *mockRepositories.MockEvaluationRubricsRepository, *mockRepositories.MockEvaluationScoresRepository, *mockRepositories.MockInterviewTurnsRepository, *mockUtils.MockGenerator) {
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockEvaluationRubricsRepo := new(mockRepositories.MockEvaluationRubricsRepository)
				mockEvaluationScoresRepo := new(mockRepositories.MockEvaluationScoresRepository)
				mockInterviewTurnsRepo := new(mockRepositories.MockInterviewTurnsRepository)
				mockGenerator := new(mockUtils.MockGenerator)

				// Mock Redis miss for multiple retries
				redisKey := fmt.Sprintf("%s%s:%s", constants.RedisPrefixInterviewIsScoreSessionState, validReq.SessionID, validReq.CurrentState)
				mockRedisClient.EXPECT().
					Get(ctx, redisKey).
					Return("", errors.New("redis miss")).
					Times(3) // Will retry 3 times before succeeding

				// Mock DB check for scored state - first few calls return false, then true
				mockEvaluationScoresRepo.EXPECT().
					IsLastUserStateTurnScored(ctx, mock.AnythingOfType("*db.IsLastUserStateTurnScoredParams")).
					Return(false, nil).
					Times(2)
				mockEvaluationScoresRepo.EXPECT().
					IsLastUserStateTurnScored(ctx, mock.AnythingOfType("*db.IsLastUserStateTurnScoredParams")).
					Return(true, nil)

				// Mock GetEvaluationSummaryJsonBySessionAndState
				mockEvaluationScoresRepo.EXPECT().
					GetEvaluationSummaryJsonBySessionAndState(ctx, mock.AnythingOfType("*db.GetEvaluationSummaryJsonBySessionAndStateParams")).
					Return(jsonEvaluationSummaryResp, nil)

				// Mock CalculateEachCriteriaCommentBySessionAndState
				mockEvaluationScoresRepo.EXPECT().
					CalculateEachCriteriaCommentBySessionAndState(ctx, mock.AnythingOfType("*repositories.PreProcessedCriteriaReq")).
					Return(validPostProcessedCriteriaResp, nil)

				// Mock generator calls
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(uuid.MustParse(validEvaluationID)).Times(1)
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(uuid.MustParse(validScoreID1)).Times(1)
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(uuid.MustParse(validScoreID2)).Times(1)

				// Mock CreatePhraseEvaluationAndCriteriaScoreWithIsScoredState
				mockEvaluationScoresRepo.EXPECT().
					CreatePhraseEvaluationAndCriteriaScoreWithIsScoredState(ctx, mock.AnythingOfType("*repositories.CreatePhraseEvaluationAndCriteriaScoreWithIsScoredStateReqTx")).
					Return(nil)

				return mockRedisClient, mockEvaluationRubricsRepo, mockEvaluationScoresRepo, mockInterviewTurnsRepo, mockGenerator
			},
			verify: func(t *testing.T, gotErr error) {
				assert.NoError(t, gotErr)
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
			)

			gotResp, gotErr := svc.GetPhraseEvaluationsWithCriteriaBySessionID(ctx, tC.input)

			tC.verify(t, gotResp, gotErr)
		})
	}
}
