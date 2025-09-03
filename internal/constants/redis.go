package constants

import "time"

// Redis Prefix Constants
const (
	RedisPrefixVerifyEmail             = "auth:verify_email:code:"
	RedisAttemptPrefixVerifyEmail      = "auth:verify_email:attempt:"
	RedisPrefixDefaultResume           = "resume:default:"
	RedisPrefixInterviewSegmentMapping = "interview:segment:mapping:"
	RedisPrefixInterviewMaxTurnNo      = "interview:max_turn_no:"
	RedisPrefixInterviewStartEndTime   = "interview:start_end_time:"

	MaxAttemptVerifyEmail = 3
	RedisTTLDefault       = 1 * time.Hour

	RedisTTLInterviewSegmentMapping = 10 * time.Minute
	RedisTTLInterviewTurn           = 10 * time.Minute
	RedisTTLInterviewStartEndTime   = 1 * time.Hour
)
