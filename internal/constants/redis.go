package constants

import "time"

// Redis Prefix Constants
const (
	RedisPrefixVerifyEmail        = "auth:verify_email:code:"
	RedisAttemptPrefixVerifyEmail = "auth:verify_email:attempt:"
	RedisPrefixDefaultResume      = "resume:default"

	MaxAttemptVerifyEmail = 3
	RedisTTLDefault       = 1 * time.Hour
)
