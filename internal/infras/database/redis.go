package database

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"gitlab.com/interview-simulation/interview-backend-server/internal/config"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
)

type RedisClient interface {
	Set(ctx context.Context, payload RedisPayload) error
	Get(ctx context.Context, key string) (string, error)
	Delete(ctx context.Context, keys ...string) error
	Increment(ctx context.Context, key string) (int64, error)
	Exists(ctx context.Context, key string) (bool, error)
	Expire(ctx context.Context, key string, duration time.Duration) error
	HSet(ctx context.Context, key string, values ...interface{}) error
	HGet(ctx context.Context, key, field string) (string, error)
	HDel(ctx context.Context, key string, fields ...string) error
	HGetAll(ctx context.Context, key string) (map[string]string, error)
}

type redisClient struct {
	client *redis.Client
	cfg    *config.Config
	log    *log.Logger
}

type RedisPayload struct {
	Key   string
	Value interface{}
	TTL   time.Duration
}
type RedisDeletePayload struct {
	Keys []string
}

func NewRedisClient(cfg *config.Config, logger *log.Logger) RedisClient {
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.RedisConfig.Host, cfg.RedisConfig.Port),
		Password: cfg.RedisConfig.Password,
		DB:       cfg.RedisConfig.DBTemp,
	})

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		logger.Errorf("Failed to connect to Redis", "error", err)
	} else {
		logger.Info("Connected to Redis successfully")
	}

	return &redisClient{
		client: rdb,
		cfg:    cfg,
		log:    logger,
	}
}

func (r *redisClient) Set(ctx context.Context, payload RedisPayload) error {

	if err := r.client.Set(ctx, payload.Key, payload.Value, payload.TTL).Err(); err != nil {
		r.log.ErrorWithID(ctx, "[Redis Client: Set] Error", err)
		return err
	}

	return nil
}

func (r *redisClient) Get(ctx context.Context, key string) (string, error) {

	res, err := r.client.Get(ctx, key).Result()
	if err != nil {
		r.log.ErrorWithID(ctx, "[Redis Client: Get] Error", err)
		return "", err
	}

	return res, nil
}

func (r *redisClient) Delete(ctx context.Context, keys ...string) error {

	if err := r.client.Del(ctx, keys...).Err(); err != nil {
		r.log.ErrorWithID(ctx, "[Redis Client: Delete] Error", err)
		return err
	}

	return nil
}

func (r *redisClient) Increment(ctx context.Context, key string) (int64, error) {
	n, err := r.client.Incr(ctx, key).Result()
	if err != nil {
		r.log.ErrorWithID(ctx, "[Redis Client: Incr] Error", err)
		return 0, err
	}
	return n, nil
}

func (r *redisClient) Exists(ctx context.Context, key string) (bool, error) {

	res, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		r.log.ErrorWithID(ctx, "[Redis Client: Exists] Error", err)
		return false, err
	}

	return res > 0, nil
}

func (r *redisClient) Expire(ctx context.Context, key string, duration time.Duration) error {

	if err := r.client.Expire(ctx, key, duration).Err(); err != nil {
		r.log.ErrorWithID(ctx, "[Redis Client: Expire] Error", err)
		return err
	}

	return nil
}

func (r *redisClient) HSet(ctx context.Context, key string, values ...interface{}) error {

	if err := r.client.HSet(ctx, key, values...).Err(); err != nil {
		r.log.ErrorWithID(ctx, "[Redis Client: HSet] Error", err)
		return err
	}

	return nil
}

func (r *redisClient) HGet(ctx context.Context, key, field string) (string, error) {

	res, err := r.client.HGet(ctx, key, field).Result()
	if err != nil {
		r.log.ErrorWithID(ctx, "[Redis Client: HGet] Error", err)
		return "", err
	}

	return res, nil
}

func (r *redisClient) HDel(ctx context.Context, key string, fields ...string) error {

	if err := r.client.HDel(ctx, key, fields...).Err(); err != nil {
		r.log.ErrorWithID(ctx, "[Redis Client: HDel] Error", err)
		return err
	}

	return nil
}

func (r *redisClient) HGetAll(ctx context.Context, key string) (map[string]string, error) {

	res, err := r.client.HGetAll(ctx, key).Result()
	if err != nil {
		r.log.ErrorWithID(ctx, "[Redis Client: HGetAll] Error", err)
		return nil, err
	}

	return res, nil
}
