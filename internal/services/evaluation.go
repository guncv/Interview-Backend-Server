package services

import (
	"context"
	"encoding/json"
	"fmt"

	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
	db "gitlab.com/interview-simulation/interview-backend-server/internal/db/sqlc"
	"gitlab.com/interview-simulation/interview-backend-server/internal/entities"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/database"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
)

type EvaluationService interface {
	GetRubricWithCriteriaByName(ctx context.Context) (*entities.GetRubricWithCriteriaByNameResp, error)
}

type evaluationService struct {
	log         *log.Logger
	db          db.Store
	redisClient database.RedisClient
}

func NewEvaluationService(
	log *log.Logger,
	db db.Store,
	redisClient database.RedisClient,
) EvaluationService {
	return &evaluationService{
		log:         log,
		db:          db,
		redisClient: redisClient,
	}
}

func (s *evaluationService) GetRubricWithCriteriaByName(ctx context.Context) (*entities.GetRubricWithCriteriaByNameResp, error) {
	s.log.InfoWithID(ctx, "[Service: GetRubricWithCriteriaByName] Called")

	var resp entities.GetRubricWithCriteriaByNameResp
	redisKey := fmt.Sprintf("%s:%s", constants.RedisPrefixEvaluationRubric, constants.RubricNameGeneralInterview)

	redisValue, err := s.redisClient.Get(ctx, redisKey)
	if err != nil {
		s.log.WarnWithID(ctx, "[Service: GetRubricWithCriteriaByName] Redis miss or error, falling back to DB", err)

		dbRows, err := s.db.GetRubricWithCriteriaByName(ctx, constants.RubricNameGeneralInterview)
		if err != nil {
			s.log.ErrorWithID(ctx, "[Service: GetRubricWithCriteriaByName] DB error after Redis fail", err)
			return nil, err
		}

		if len(dbRows) == 0 {
			s.log.ErrorWithID(ctx, "[Service: GetRubricWithCriteriaByName] No rubric/criteria found in DB")
			return nil, app_error.New(err, app_error.ErrCodeEvaluationRubricCriteriaNotFound)
		}

		criteria := make([]entities.CritetiaRow, 0, len(dbRows))
		for _, row := range dbRows {
			criteria = append(criteria, entities.CritetiaRow{
				CriterionID:            row.CriterionID.String(),
				CriterionCode:          row.CriterionCode,
				CriterionName:          row.CriterionName,
				CriterionDescriptionMd: row.CriterionDescriptionMd.String,
				CriterionWeight:        row.CriterionWeight,
				CriterionMaxScore:      row.CriterionMaxScore,
			})
		}

		resp = entities.GetRubricWithCriteriaByNameResp{
			RubricID:            dbRows[0].RubricID.String(),
			RubricName:          dbRows[0].RubricName,
			RubricDescriptionMd: dbRows[0].RubricDescriptionMd.String,
			RubricVersionLabel:  dbRows[0].RubricVersionLabel.String,
			Criteria:            criteria,
		}

		go func() {
			if jsonBytes, err := json.Marshal(resp); err == nil {
				_ = s.redisClient.Set(ctx, database.RedisPayload{
					Key:   redisKey,
					Value: jsonBytes,
					TTL:   constants.RedisTTLEvaluationRubric,
				})
			}
		}()

		return &resp, nil
	}

	s.log.InfoWithID(ctx, "[Service: GetRubricWithCriteriaByName] Cache hit")
	if err := json.Unmarshal([]byte(redisValue), &resp); err != nil {
		s.log.ErrorWithID(ctx, "[Service: GetRubricWithCriteriaByName] Failed to unmarshal Redis value", err)
		return nil, app_error.New(err, app_error.ErrCodeGeneralUnmarshalFailed)
	}

	return &resp, nil
}
