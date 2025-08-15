package queue

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
	"gitlab.com/interview-simulation/interview-backend-server/internal/config"
	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/email"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
)

type RedisTaskPublisher interface {
	PublishTaskSendResetPasswordEmail(ctx context.Context, payload *email.ResetPasswordEmailPayload, opts ...asynq.Option) error
	DefineTaskOptions(taskName string) []asynq.Option
}

type redisTaskPublisher struct {
	client *asynq.Client
	log    *log.Logger
}

func NewRedisTaskPublisher(cfg *config.Config, log *log.Logger) RedisTaskPublisher {
	rdb := asynq.RedisClientOpt{
		Addr:     fmt.Sprintf("%s:%s", cfg.RedisConfig.Host, cfg.RedisConfig.Port),
		Password: cfg.RedisConfig.Password,
		DB:       cfg.RedisConfig.DBQueue,
	}
	client := asynq.NewClient(rdb)

	return &redisTaskPublisher{
		client: client,
		log:    log,
	}
}

func (p *redisTaskPublisher) PublishTaskSendResetPasswordEmail(ctx context.Context, payload *email.ResetPasswordEmailPayload, opts ...asynq.Option) error {
	p.log.InfoWithID(ctx, "[Queue: PublishTaskSendResetPasswordEmail] Called")
	jsonPayload, err := json.Marshal(payload)

	if err != nil {
		p.log.ErrorWithID(ctx, "[Queue: PublishTaskSendResetPasswordEmail] Error marshalling task payload", err)
		return app_error.New(err, app_error.ErrCodeGeneralServerUnavailable)
	}

	task := asynq.NewTask(constants.TaskSendResetPasswordEmail, jsonPayload, opts...)
	info, err := p.client.EnqueueContext(ctx, task)
	if err != nil {
		p.log.ErrorWithID(ctx, "[Queue: PublishTaskSendResetPasswordEmail] Error enqueuing task", err)
		return app_error.New(err, app_error.ErrCodeGeneralServerUnavailable)
	}

	p.log.InfoWithID(ctx, "[Queue: PublishTaskSendResetPasswordEmail] Enqueued task", info)
	return nil
}

func (p *redisTaskPublisher) DefineTaskOptions(taskName string) []asynq.Option {
	switch taskName {
	case constants.TaskSendResetPasswordEmail:
		return []asynq.Option{
			asynq.MaxRetry(constants.MaxRetry),
			asynq.Queue(constants.QueueCritical),
		}
	}
	return []asynq.Option{
		asynq.MaxRetry(constants.MaxRetry),
		asynq.Queue(constants.QueueCritical),
	}
}
