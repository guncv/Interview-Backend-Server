package publisher

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
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/email"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
)

type RedisTaskPublisher interface {
	PublishTaskSendResetPasswordEmail(ctx context.Context, payload *email.ResetPasswordEmailPayload, opts ...asynq.Option) error
	PublishTaskSendVerifyEmail(ctx context.Context, payload *email.VerifyEmailPayload, opts ...asynq.Option) error
	PublishTaskDeleteFile(ctx context.Context, payload *aws.DeleteFilePayload, opts ...asynq.Option) error
	PublishTaskCalculateTurnScore(ctx context.Context, payload *entities.CalculateTurnScoreReq, opts ...asynq.Option) error
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

func (p *redisTaskPublisher) PublishTaskSendVerifyEmail(ctx context.Context, payload *email.VerifyEmailPayload, opts ...asynq.Option) error {
	p.log.InfoWithID(ctx, "[Queue: PublishTaskSendVerifyEmail] Called")
	jsonPayload, err := json.Marshal(payload)

	if err != nil {
		p.log.ErrorWithID(ctx, "[Queue: PublishTaskSendVerifyEmail] Error marshalling task payload", err)
		return app_error.New(err, app_error.ErrCodeGeneralServerUnavailable)
	}

	task := asynq.NewTask(constants.TaskSendVerifyEmail, jsonPayload, opts...)
	info, err := p.client.EnqueueContext(ctx, task)
	if err != nil {
		p.log.ErrorWithID(ctx, "[Queue: PublishTaskSendVerifyEmail] Error enqueuing task", err)
		return app_error.New(err, app_error.ErrCodeGeneralServerUnavailable)
	}

	p.log.InfoWithID(ctx, "[Queue: PublishTaskSendVerifyEmail] Enqueued task", info)
	return nil
}

func (p *redisTaskPublisher) PublishTaskDeleteFile(ctx context.Context, payload *aws.DeleteFilePayload, opts ...asynq.Option) error {
	p.log.InfoWithID(ctx, "[Queue: PublishTaskDeleteFile] Called")
	jsonPayload, err := json.Marshal(payload)

	if err != nil {
		p.log.ErrorWithID(ctx, "[Queue: PublishTaskDeleteFile] Error marshalling task payload", err)
		return app_error.New(err, app_error.ErrCodeGeneralServerUnavailable)
	}

	task := asynq.NewTask(constants.TaskDeleteFile, jsonPayload, opts...)
	info, err := p.client.EnqueueContext(ctx, task)
	if err != nil {
		p.log.ErrorWithID(ctx, "[Queue: PublishTaskDeleteFile] Error enqueuing task", err)
		return app_error.New(err, app_error.ErrCodeGeneralServerUnavailable)
	}

	p.log.InfoWithID(ctx, "[Queue: PublishTaskDeleteFile] Enqueued task", info)
	return nil
}

func (p *redisTaskPublisher) PublishTaskCalculateTurnScore(ctx context.Context, payload *entities.CalculateTurnScoreReq, opts ...asynq.Option) error {
	p.log.InfoWithID(ctx, "[Queue: PublishTaskCalculateTurnScore] Called")
	jsonPayload, err := json.Marshal(payload)

	if err != nil {
		p.log.ErrorWithID(ctx, "[Queue: PublishTaskCalculateTurnScore] Error marshalling task payload", err)
		return app_error.New(err, app_error.ErrCodeGeneralServerUnavailable)
	}

	task := asynq.NewTask(constants.TaskCalculateTurnScore, jsonPayload, opts...)
	info, err := p.client.EnqueueContext(ctx, task)
	if err != nil {
		p.log.ErrorWithID(ctx, "[Queue: PublishTaskCalculateTurnScore] Error enqueuing task", err)
		return app_error.New(err, app_error.ErrCodeGeneralServerUnavailable)
	}

	p.log.InfoWithID(ctx, "[Queue: PublishTaskDeleteRedis] Enqueued task", info)
	return nil
}

func (p *redisTaskPublisher) DefineTaskOptions(taskName string) []asynq.Option {
	switch taskName {
	case constants.TaskSendResetPasswordEmail:
		return []asynq.Option{
			asynq.MaxRetry(constants.MaxRetry),
			asynq.Queue(constants.QueueCritical),
		}
	case constants.TaskCalculateTurnScore:
		return []asynq.Option{
			asynq.MaxRetry(constants.MaxRetry),
			asynq.Queue(constants.QueueCritical),
		}
	case constants.TaskSendVerifyEmail:
		return []asynq.Option{
			asynq.MaxRetry(constants.MaxRetry),
			asynq.Queue(constants.QueueCritical),
		}
	case constants.TaskDeleteFile:
		return []asynq.Option{
			asynq.MaxRetry(constants.MaxRetry),
			asynq.Queue(constants.QueueDefault),
		}
	case constants.TaskSetRedis:
		return []asynq.Option{
			asynq.MaxRetry(constants.MaxRetry),
			asynq.Queue(constants.QueueDefault),
		}
	case constants.TaskDeleteRedis:
		return []asynq.Option{
			asynq.MaxRetry(constants.MaxRetry),
			asynq.Queue(constants.QueueDefault),
		}
	}
	return []asynq.Option{
		asynq.MaxRetry(constants.MaxRetry),
		asynq.Queue(constants.QueueDefault),
	}
}
