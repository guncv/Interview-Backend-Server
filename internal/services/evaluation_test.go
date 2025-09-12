package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
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
					Set(ctx, database.RedisPayload{
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
