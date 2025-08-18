package queue

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
	"gitlab.com/interview-simulation/interview-backend-server/internal/config"
	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/aws"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/database"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/email"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
)

type RedisTaskConsumer interface {
	Start(ctx context.Context) error
	ConsumeTaskSendResetPasswordEmail(ctx context.Context, task *asynq.Task) error
	ConsumeTaskSendVerifyEmail(ctx context.Context, task *asynq.Task) error
	ConsumeTaskDeleteFile(ctx context.Context, task *asynq.Task) error
	ConsumeTaskSetRedis(ctx context.Context, task *asynq.Task) error
	ConsumeTaskDeleteRedis(ctx context.Context, task *asynq.Task) error
}

type redisTaskConsumer struct {
	server      *asynq.Server
	log         *log.Logger
	emailSender email.EmailSender
	s3Storage   aws.S3Storage
	redisClient database.RedisClient
}

func NewRedisTaskConsumer(
	cfg *config.Config,
	log *log.Logger,
	emailSender email.EmailSender,
	s3Storage aws.S3Storage,
	redisClient database.RedisClient,
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
		server:      server,
		log:         log,
		emailSender: emailSender,
		s3Storage:   s3Storage,
		redisClient: redisClient,
	}
}

func (c *redisTaskConsumer) Start(ctx context.Context) error {
	c.log.InfoWithID(ctx, "[Email: Start] Starting email consumer server")
	mux := asynq.NewServeMux()
	mux.HandleFunc(constants.TaskSendResetPasswordEmail, c.ConsumeTaskSendResetPasswordEmail)
	mux.HandleFunc(constants.TaskSendVerifyEmail, c.ConsumeTaskSendVerifyEmail)
	mux.HandleFunc(constants.TaskDeleteFile, c.ConsumeTaskDeleteFile)
	mux.HandleFunc(constants.TaskSetRedis, c.ConsumeTaskSetRedis)
	mux.HandleFunc(constants.TaskDeleteRedis, c.ConsumeTaskDeleteRedis)

	if err := c.server.Start(mux); err != nil {
		c.log.ErrorWithID(ctx, "[Email: Start] Failed to start server", err)
		return fmt.Errorf("failed to start email consumer server: %w", err)
	}

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

func (c *redisTaskConsumer) ConsumeTaskSetRedis(ctx context.Context, task *asynq.Task) error {
	c.log.InfoWithID(ctx, "[Email: ConsumeTaskSetRedis] Processing set redis task")

	var payload database.RedisPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		c.log.ErrorWithID(ctx, "[Email: ConsumeTaskSetRedis] Failed to unmarshal payload", err)
		return app_error.New(fmt.Errorf("invalid set redis payload: %w", err), app_error.ErrCodeGeneralServerUnavailable)
	}

	if err := c.redisClient.Set(ctx, payload); err != nil {
		c.log.ErrorWithID(ctx, "[Email: ConsumeTaskSetRedis] Failed to set redis", err)
		return app_error.New(fmt.Errorf("failed to set redis: %w", err), app_error.ErrCodeGeneralServerUnavailable)
	}

	c.log.InfoWithID(ctx, "[Email: ConsumeTaskSetRedis] Successfully set redis", nil)
	return nil
}

func (c *redisTaskConsumer) ConsumeTaskDeleteRedis(ctx context.Context, task *asynq.Task) error {
	c.log.InfoWithID(ctx, "[Email: ConsumeTaskDeleteRedis] Processing delete redis task")

	var payload database.RedisDeletePayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		c.log.ErrorWithID(ctx, "[Email: ConsumeTaskDeleteRedis] Failed to unmarshal payload", err)
		return app_error.New(fmt.Errorf("invalid delete redis payload: %w", err), app_error.ErrCodeGeneralServerUnavailable)
	}

	for _, key := range payload.Keys {
		if err := c.redisClient.Delete(ctx, key); err != nil {
			c.log.ErrorWithID(ctx, "[Email: ConsumeTaskDeleteRedis] Failed to delete redis key", err)
			return app_error.New(fmt.Errorf("failed to delete redis key: %w", err), app_error.ErrCodeGeneralServerUnavailable)
		}
	}

	c.log.InfoWithID(ctx, "[Email: ConsumeTaskDeleteRedis] Successfully deleted redis", nil)
	return nil
}
