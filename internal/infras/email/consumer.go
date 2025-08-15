package email

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
	"gitlab.com/interview-simulation/interview-backend-server/internal/config"
	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
)

const (
	QueueCritical = "critical"
	QueueDefault  = "default"
)

type RedisTaskConsumer interface {
	Start(ctx context.Context) error
	ConsumeTaskSendResetPasswordEmail(ctx context.Context, task *asynq.Task) error
}

type redisTaskConsumer struct {
	server      *asynq.Server
	log         *log.Logger
	emailSender EmailSender
}

func NewRedisTaskConsumer(
	cfg *config.Config,
	log *log.Logger,
	emailSender EmailSender,
) RedisTaskConsumer {
	redisOpt := asynq.RedisClientOpt{
		Addr:     fmt.Sprintf("%s:%s", cfg.RedisConfig.Host, cfg.RedisConfig.Port),
		Password: cfg.RedisConfig.Password,
		DB:       cfg.RedisConfig.DBQueue,
	}

	server := asynq.NewServer(redisOpt, asynq.Config{
		Concurrency: 10,
		Queues: map[string]int{
			QueueCritical: 10,
			QueueDefault:  5,
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
	}
}

func (c *redisTaskConsumer) Start(ctx context.Context) error {
	c.log.InfoWithID(ctx, "[Email: Start] Called")
	mux := asynq.NewServeMux()
	mux.HandleFunc(constants.TaskSendResetPasswordEmail, c.ConsumeTaskSendResetPasswordEmail)

	if err := c.server.Start(mux); err != nil {
		c.log.ErrorWithID(ctx, "[Email: Start] Error starting server", err)
		return err
	}

	return nil
}

func (c *redisTaskConsumer) ConsumeTaskSendResetPasswordEmail(ctx context.Context, task *asynq.Task) error {
	c.log.InfoWithID(ctx, "[Email: ComsumeTaskSendResetPasswordEmail] Called")
	var payload ResetPasswordEmailPayload

	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		c.log.ErrorWithID(ctx, "[Email: ComsumeTaskSendResetPasswordEmail] Error unmarshalling payload", err)
		return app_error.New(err, app_error.ErrCodeGeneralServerUnavailable)
	}

	if err := c.emailSender.SendResetPasswordEmail(ctx, &payload); err != nil {
		c.log.ErrorWithID(ctx, "[Email: ComsumeTaskSendResetPasswordEmail] Error sending reset password email", err)
		return app_error.New(err, app_error.ErrCodeGeneralServerUnavailable)
	}

	return nil
}
