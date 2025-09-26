package consumer

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
	"gitlab.com/interview-simulation/interview-backend-server/internal/config"
	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
	"gitlab.com/interview-simulation/interview-backend-server/internal/entities"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/aws"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/database"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/email"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
	"gitlab.com/interview-simulation/interview-backend-server/internal/services"
)

type RedisTaskConsumer interface {
	Start(ctx context.Context) error
	CleanupQueue(ctx context.Context) error
	ConsumeTaskSendResetPasswordEmail(ctx context.Context, task *asynq.Task) error
	ConsumeTaskSendVerifyEmail(ctx context.Context, task *asynq.Task) error
	ConsumeTaskDeleteFile(ctx context.Context, task *asynq.Task) error
	ConsumeTaskCalculateTurnScore(ctx context.Context, task *asynq.Task) error
}

type redisTaskConsumer struct {
	server                  *asynq.Server
	log                     *log.Logger
	emailSender             email.EmailSender
	s3Storage               aws.S3Storage
	redisClient             database.RedisClient
	interviewSessionService services.InterviewSessionService
	evaluationService       services.EvaluationService
}

func NewRedisTaskConsumer(
	cfg *config.Config,
	log *log.Logger,
	emailSender email.EmailSender,
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
		emailSender:             emailSender,
		s3Storage:               s3Storage,
		redisClient:             redisClient,
		interviewSessionService: interviewSessionService,
		evaluationService:       evaluationService,
	}
}

func (c *redisTaskConsumer) Start(ctx context.Context) error {
	c.log.InfoWithID(ctx, "[Email: Start] Starting email consumer server")
	mux := asynq.NewServeMux()
	mux.HandleFunc(constants.TaskSendResetPasswordEmail, c.ConsumeTaskSendResetPasswordEmail)
	mux.HandleFunc(constants.TaskSendVerifyEmail, c.ConsumeTaskSendVerifyEmail)
	mux.HandleFunc(constants.TaskDeleteFile, c.ConsumeTaskDeleteFile)
	mux.HandleFunc(constants.TaskCalculateTurnScore, c.ConsumeTaskCalculateTurnScore)

	if err := c.server.Start(mux); err != nil {
		c.log.ErrorWithID(ctx, "[Email: Start] Failed to start server", err)
		return fmt.Errorf("failed to start email consumer server: %w", err)
	}

	return nil
}

func (c *redisTaskConsumer) CleanupQueue(ctx context.Context) error {
	c.log.InfoWithID(ctx, "[Email: CleanupQueue] Cleaning up email queue")

	if err := c.redisClient.Delete(ctx, constants.QueueDefault); err != nil {
		c.log.WarnWithID(ctx, "Failed to cleanup default queue", err)
	}

	if err := c.redisClient.Delete(ctx, constants.QueueCritical); err != nil {
		c.log.WarnWithID(ctx, "Failed to cleanup critical queue", err)
	}

	c.log.InfoWithID(ctx, "[Email: CleanupQueue] Queue cleanup completed")
	return nil
}

func (c *redisTaskConsumer) ConsumeTaskSendResetPasswordEmail(ctx context.Context, task *asynq.Task) error {
	c.log.InfoWithID(ctx, "[Email: ConsumeTaskSendResetPasswordEmail] Processing reset password email task")

	var payload email.ResetPasswordEmailPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		c.log.ErrorWithID(ctx, "[Email: ConsumeTaskSendResetPasswordEmail] Failed to unmarshal payload", err)
		return app_error.New(fmt.Errorf("invalid reset password email payload: %w", err), app_error.ErrCodeGeneralServerUnavailable)
	}

	if err := c.emailSender.SendResetPasswordEmail(ctx, &payload); err != nil {
		c.log.ErrorWithID(ctx, "[Email: ConsumeTaskSendResetPasswordEmail] Failed to send reset password email", err)
		return app_error.New(fmt.Errorf("failed to send reset password email: %w", err), app_error.ErrCodeGeneralServerUnavailable)
	}

	c.log.InfoWithID(ctx, "[Email: ConsumeTaskSendResetPasswordEmail] Successfully sent reset password email", nil)
	return nil
}

func (c *redisTaskConsumer) ConsumeTaskSendVerifyEmail(ctx context.Context, task *asynq.Task) error {
	c.log.InfoWithID(ctx, "[Email: ConsumeTaskSendVerifyEmail] Processing verify email task")

	var payload email.VerifyEmailPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		c.log.ErrorWithID(ctx, "[Email: ConsumeTaskSendVerifyEmail] Failed to unmarshal payload", err)
		return app_error.New(fmt.Errorf("invalid verify email payload: %w", err), app_error.ErrCodeGeneralServerUnavailable)
	}

	if err := c.emailSender.SendVerifyEmail(ctx, &payload); err != nil {
		c.log.ErrorWithID(ctx, "[Email: ConsumeTaskSendVerifyEmail] Failed to send verify email", err)
		return app_error.New(fmt.Errorf("failed to send verify email: %w", err), app_error.ErrCodeGeneralServerUnavailable)
	}

	c.log.InfoWithID(ctx, "[Email: ConsumeTaskSendVerifyEmail] Successfully sent verify email", nil)
	return nil
}

func (c *redisTaskConsumer) ConsumeTaskDeleteFile(ctx context.Context, task *asynq.Task) error {
	c.log.InfoWithID(ctx, "[Email: ConsumeTaskDeleteFile] Processing delete file task")

	var payload aws.DeleteFilePayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		c.log.ErrorWithID(ctx, "[Email: ConsumeTaskDeleteFile] Failed to unmarshal payload", err)
		return app_error.New(fmt.Errorf("invalid delete file payload: %w", err), app_error.ErrCodeGeneralServerUnavailable)
	}

	if err := c.s3Storage.DeleteFile(ctx, payload.Key); err != nil {
		c.log.ErrorWithID(ctx, "[Email: ConsumeTaskDeleteFile] Failed to delete file", err)
		return app_error.New(fmt.Errorf("failed to delete file: %w", err), app_error.ErrCodeGeneralServerUnavailable)
	}

	c.log.InfoWithID(ctx, "[Email: ConsumeTaskDeleteFile] Successfully deleted file", nil)
	return nil
}

func (c *redisTaskConsumer) ConsumeTaskCalculateTurnScore(ctx context.Context, task *asynq.Task) error {
	c.log.InfoWithID(ctx, "[Email: ConsumeTaskCalculateTurnScore] Processing calculate turn score task")

	var payload entities.CalculateTurnScoreReq
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		c.log.ErrorWithID(ctx, "[Email: ConsumeTaskCalculateTurnScore] Failed to unmarshal payload", err)
		return app_error.New(fmt.Errorf("invalid calculate turn score payload: %w", err), app_error.ErrCodeGeneralServerUnavailable)
	}

	if err := c.evaluationService.CalculateTurnScore(ctx, &payload); err != nil {
		c.log.ErrorWithID(ctx, "[Email: ConsumeTaskCalculateTurnScore] Failed to calculate turn score", err)
		return app_error.New(fmt.Errorf("failed to calculate turn score: %w", err), app_error.ErrCodeGeneralServerUnavailable)
	}

	c.log.InfoWithID(ctx, "[Email: ConsumeTaskCalculateTurnScore] Successfully deleted redis", nil)
	return nil
}
