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
			)

			gotResp, gotErr := svc.ListAllRubricsAndCriteria(ctx)

			tC.verify(t, gotResp, gotErr)
		})
	}
}
