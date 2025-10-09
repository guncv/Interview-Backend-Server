package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"gitlab.com/interview-simulation/interview-backend-server/internal/config"
	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
	db "gitlab.com/interview-simulation/interview-backend-server/internal/db/sqlc"
	"gitlab.com/interview-simulation/interview-backend-server/internal/entities"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/aws"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/database"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
	"gitlab.com/interview-simulation/interview-backend-server/internal/services"
)

type RedisTaskConsumer interface {
	Start(ctx context.Context) error
	CleanupQueue(ctx context.Context) error
	ConsumeTaskDeleteFile(ctx context.Context, task *asynq.Task) error
	ConsumeTaskCalculateTurnScore(ctx context.Context, task *asynq.Task) error
	ConsumeTaskEndInterviewSession(ctx context.Context, task *asynq.Task) error
}

type redisTaskConsumer struct {
	server                  *asynq.Server
	log                     *log.Logger
	s3Storage               aws.S3Storage
	redisClient             database.RedisClient
	interviewSessionService services.InterviewSessionService
	evaluationService       services.EvaluationService
}

func NewRedisTaskConsumer(
	cfg *config.Config,
	log *log.Logger,
	s3Storage aws.S3Storage,
	redisClient database.RedisClient,
	interviewSessionService services.InterviewSessionService,
	evaluationService services.EvaluationService,
) RedisTaskConsumer {
	redisOpt := asynq.RedisClientOpt{
		Addr:     fmt.Sprintf("%s:%s", cfg.RedisConfig.Host, cfg.RedisConfig.Port),
		Password: cfg.RedisConfig.Password,
		DB:       cfg.RedisConfig.DBQueue,
	}

	server := asynq.NewServer(redisOpt, asynq.Config{
		Concurrency: constants.DefaultConcurrency,
		Queues: map[string]int{
			constants.QueueCritical: constants.CriticalQueueConcurrency,
			constants.QueueDefault:  constants.DefaultQueueConcurrency,
		},
		ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
			log.ErrorWithID(ctx, "[Email: Consumer] Error processing task", err)
		}),
		Logger: log,
	})

	return &redisTaskConsumer{
		server:                  server,
		log:                     log,
		s3Storage:               s3Storage,
		redisClient:             redisClient,
		interviewSessionService: interviewSessionService,
		evaluationService:       evaluationService,
	}
}

func (c *redisTaskConsumer) Start(ctx context.Context) error {
	mux := asynq.NewServeMux()
	mux.HandleFunc(constants.TaskDeleteFile, c.ConsumeTaskDeleteFile)
	mux.HandleFunc(constants.TaskCalculateTurnScore, c.ConsumeTaskCalculateTurnScore)
	mux.HandleFunc(constants.TaskEndInterviewSession, c.ConsumeTaskEndInterviewSession)

	if err := c.server.Start(mux); err != nil {
		c.log.ErrorWithID(ctx, "[Email: Start] Failed to start server", err)
		return fmt.Errorf("failed to start email consumer server: %w", err)
	}

	return nil
}

func (c *redisTaskConsumer) CleanupQueue(ctx context.Context) error {

	if err := c.redisClient.Delete(ctx, constants.QueueDefault); err != nil {
		c.log.WarnWithID(ctx, "Failed to cleanup default queue", err)
	}

	if err := c.redisClient.Delete(ctx, constants.QueueCritical); err != nil {
		c.log.WarnWithID(ctx, "Failed to cleanup critical queue", err)
	}

	return nil
}

func (c *redisTaskConsumer) ConsumeTaskDeleteFile(ctx context.Context, task *asynq.Task) error {

	var payload aws.DeleteFilePayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		c.log.ErrorWithID(ctx, "[Email: ConsumeTaskDeleteFile] Failed to unmarshal payload", err)
		return app_error.New(fmt.Errorf("invalid delete file payload: %w", err), app_error.ErrCodeGeneralServerUnavailable)
	}

	if err := c.s3Storage.DeleteFile(ctx, payload.Key); err != nil {
		c.log.ErrorWithID(ctx, "[Email: ConsumeTaskDeleteFile] Failed to delete file", err)
		return app_error.New(fmt.Errorf("failed to delete file: %w", err), app_error.ErrCodeGeneralServerUnavailable)
	}

	return nil
}

func (c *redisTaskConsumer) ConsumeTaskCalculateTurnScore(ctx context.Context, task *asynq.Task) error {

	var payload entities.CalculateTurnScoreReq
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		c.log.ErrorWithID(ctx, "[Email: ConsumeTaskCalculateTurnScore] Failed to unmarshal payload", err)
		_, _ = c.redisClient.Decrement(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewPendingScores, payload.SessionID))
		return app_error.New(fmt.Errorf("invalid calculate turn score payload: %w", err), app_error.ErrCodeGeneralServerUnavailable)
	}

	if err := c.evaluationService.CalculateTurnScore(ctx, &payload); err != nil {
		c.log.ErrorWithID(ctx, "[Email: ConsumeTaskCalculateTurnScore] Failed to calculate turn score", err)
		_, _ = c.redisClient.Decrement(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewPendingScores, payload.SessionID))
		return app_error.New(fmt.Errorf("failed to calculate turn score: %w", err), app_error.ErrCodeGeneralServerUnavailable)
	}

	_, _ = c.redisClient.Decrement(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewPendingScores, payload.SessionID))
	return nil
}

func (c *redisTaskConsumer) ConsumeTaskEndInterviewSession(ctx context.Context, task *asynq.Task) error {

	var payload entities.EndInterviewSessionPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		c.log.ErrorWithID(ctx, "[Consumer: ConsumeTaskEndInterviewSession] Failed to unmarshal payload", err)
		return app_error.New(fmt.Errorf("invalid end interview session payload: %w", err), app_error.ErrCodeGeneralServerUnavailable)
	}

	if err := c.evaluationService.UpdateFinalizeStatusSessionEvaluation(ctx, payload.SessionID, db.FinalizeStatusEnumFinalizing); err != nil {
		c.log.ErrorWithID(ctx, "[Consumer: ConsumeTaskEndInterviewSession] Failed to finalize session phrase evaluation", err)
		return app_error.New(fmt.Errorf("failed to finalize session phrase evaluation: %w", err), app_error.ErrCodeGeneralServerUnavailable)
	}

	pendingScoresKey := fmt.Sprintf("%s%s", constants.RedisPrefixInterviewPendingScores, payload.SessionID)
	maxRetries := constants.MaxRetryEndInterviewSession
	retryDelay := time.Duration(constants.RetryDelayEndInterviewSession) * time.Second

	for attempt := 0; attempt < maxRetries; attempt++ {
		pendingScoresStr, _ := c.redisClient.Get(ctx, pendingScoresKey)

		if pendingScoresStr == "0" {
			break
		}

		time.Sleep(retryDelay)
	}

	endInterviewReq := &entities.EndInterviewSessionReq{
		SessionId: payload.SessionID,
		Status:    payload.Status,
	}

	if err := c.interviewSessionService.EndInterviewSession(ctx, endInterviewReq); err != nil {
		c.log.ErrorWithID(ctx, "[Consumer: ConsumeTaskEndInterviewSession] Failed to end interview session", err)
		return app_error.New(fmt.Errorf("failed to end interview session: %w", err), app_error.ErrCodeGeneralServerUnavailable)
	}

	return nil
}
