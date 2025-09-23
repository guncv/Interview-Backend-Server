package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
	db "gitlab.com/interview-simulation/interview-backend-server/internal/db/sqlc"
	"gitlab.com/interview-simulation/interview-backend-server/internal/entities"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/database"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
	"gitlab.com/interview-simulation/interview-backend-server/internal/repositories"
)

type EvaluationService interface {
	GetRubricWithCriteriaByName(ctx context.Context, rubricName string) (*entities.GetRubricWithCriteriaByNameResp, error)
	ListAllRubricsAndCriteria(ctx context.Context) (*entities.ListAllRubricsAndCriteriaResp, error)
}

type evaluationService struct {
	log                   *log.Logger
	redisClient           database.RedisClient
	evaluationRubricsRepo repositories.EvaluationRubricsRepository
}

func NewEvaluationService(
	log *log.Logger,
	redisClient database.RedisClient,
	evaluationRubricsRepo repositories.EvaluationRubricsRepository,
) EvaluationService {
	return &evaluationService{
		log:                   log,
		evaluationRubricsRepo: evaluationRubricsRepo,
		redisClient:           redisClient,
	}
}

func (s *evaluationService) GetRubricWithCriteriaByName(ctx context.Context, rubricName string) (*entities.GetRubricWithCriteriaByNameResp, error) {
	s.log.InfoWithID(ctx, "[Service: GetRubricWithCriteriaByName] Called")

	var resp entities.GetRubricWithCriteriaByNameResp
	redisKey := fmt.Sprintf("%s:%s", constants.RedisPrefixEvaluationRubric, rubricName)

	redisValue, err := s.redisClient.Get(ctx, redisKey)
	if err != nil {
		s.log.WarnWithID(ctx, "[Service: GetRubricWithCriteriaByName] Redis miss or error, falling back to DB", err)

		criteriaNameMapping := constants.StateToCriteriaStateMap[rubricName]
		req := &db.GetRubricWithCriteriaByNameParams{
			Name:         criteriaNameMapping,
			VersionLabel: constants.CurrentCriteriaVersion,
		}

		dbRows, err := s.evaluationRubricsRepo.GetRubricWithCriteriaByName(ctx, req)
		if err != nil {
			s.log.ErrorWithID(ctx, "[Service: GetRubricWithCriteriaByName] DB error after Redis fail", err)
			return nil, err
		}

		if len(dbRows) == 0 {
			s.log.ErrorWithID(ctx, "[Service: GetRubricWithCriteriaByName] No rubric/criteria found in DB")
			return nil, app_error.New(errors.New("no rubric/criteria found"), app_error.ErrCodeEvaluationRubricCriteriaNotFound)
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
			RubricVersionLabel:  dbRows[0].RubricVersionLabel,
			Criteria:            criteria,
		}

		go func() {
			cacheCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			if jsonBytes, err := json.Marshal(resp); err == nil {
				_ = s.redisClient.Set(cacheCtx, database.RedisPayload{
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

func (s *evaluationService) ListAllRubricsAndCriteria(ctx context.Context) (*entities.ListAllRubricsAndCriteriaResp, error) {
	s.log.InfoWithID(ctx, "[Service: ListAllRubricsAndCriteria] Called")

	var resp entities.ListAllRubricsAndCriteriaResp
	redisKey := constants.RedisPrefixAllRubricsAndCriteria

	redisValue, err := s.redisClient.Get(ctx, redisKey)
	if err != nil {
		s.log.WarnWithID(ctx, "[Service: ListAllRubricsAndCriteria] Redis miss or error, falling back to DB", err)

		resp, err := s.fetchAllRubricsAndCriteriaFromDB(ctx)
		if err != nil {
			s.log.ErrorWithID(ctx, "[Service: ListAllRubricsAndCriteria] Error getting all rubrics and criteria", err)
			return nil, err
		}
		return resp, nil
	}

	if err := json.Unmarshal([]byte(redisValue), &resp); err != nil {
		s.log.WarnWithID(ctx, "[Service: ListAllRubricsAndCriteria] Failed to unmarshal Redis value, falling back to DB", err)

		resp, err := s.fetchAllRubricsAndCriteriaFromDB(ctx)
		if err != nil {
			s.log.ErrorWithID(ctx, "[Service: ListAllRubricsAndCriteria] Error getting all rubrics and criteria", err)
			return nil, err
		}
		return resp, nil
	}

	return &resp, nil
}

func (s *evaluationService) fetchAllRubricsAndCriteriaFromDB(ctx context.Context) (*entities.ListAllRubricsAndCriteriaResp, error) {
	s.log.InfoWithID(ctx, "[Service: fetchAllRubricsAndCriteriaFromDB] Fetching all rubrics and criteria from DB")

	redisKey := constants.RedisPrefixAllRubricsAndCriteria

	dbRows, err := s.evaluationRubricsRepo.ListAllRubricsAndCriteria(ctx, constants.CurrentCriteriaVersion)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: fetchAllRubricsAndCriteriaFromDB] Error getting all rubrics and criteria", err)
		return nil, err
	}

	if len(dbRows) == 0 {
		s.log.InfoWithID(ctx, "[Service: fetchAllRubricsAndCriteriaFromDB] No rubrics found")

		resp := entities.ListAllRubricsAndCriteriaResp{
			Rubrics: []entities.RubricAndCriteriaRow{},
		}

		return &resp, nil
	}

	rubricMap := make(map[string]*entities.RubricAndCriteriaRow)

	for _, row := range dbRows {
		rubricID := row.RubricID.String()

		if _, exists := rubricMap[rubricID]; !exists {
			rubricMap[rubricID] = &entities.RubricAndCriteriaRow{
				ID:            row.RubricID.String(),
				Name:          row.RubricName,
				DescriptionMd: row.RubricDescriptionMd.String,
				VersionLabel:  row.RubricVersionLabel,
				Criteria:      []entities.CriterionRow{},
			}
		}

		criterion := entities.CriterionRow{
			ID:            row.CriterionID.String(),
			Name:          row.CriterionName,
			DescriptionMd: row.CriterionDescriptionMd.String,
			Weight:        row.CriterionWeight,
			MaxScore:      row.CriterionMaxScore,
		}

		rubricMap[rubricID].Criteria = append(rubricMap[rubricID].Criteria, criterion)
	}

	result := make([]entities.RubricAndCriteriaRow, 0, len(rubricMap))
	for _, rubricAndCriteria := range rubricMap {
		result = append(result, *rubricAndCriteria)
	}

	resp := entities.ListAllRubricsAndCriteriaResp{
		Rubrics: result,
	}

	go func() {
		cacheCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		if jsonBytes, err := json.Marshal(resp); err == nil {
			_ = s.redisClient.Set(cacheCtx, database.RedisPayload{
				Key:   redisKey,
				Value: jsonBytes,
				TTL:   constants.RedisTTLEvaluationRubric,
			})
		}
	}()

	return &resp, nil
}
