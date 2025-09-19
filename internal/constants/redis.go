package constants

import "time"

// Redis Prefix Constants
const (
	RedisPrefixVerifyEmail                 = "auth:verify_email:code:"
	RedisAttemptPrefixVerifyEmail          = "auth:verify_email:attempt:"
	RedisPrefixDefaultResume               = "resume:default:"
	RedisPrefixInterviewSegmentMapping     = "interview:segment:mapping:"
	RedisPrefixInterviewMaxTurnNo          = "interview:max_turn_no:"
	RedisPrefixInterviewStartEndTime       = "interview:start_end_time:"
	RedisPrefixInterviewLastMessage        = "interview:last_message:"
	RedisPrefixEvaluationRubric            = "evaluation:rubric:general:"
	RedisPrefixInterviewSessionToken       = "interview:session_token:"
	RedisPrefixInterviewSessionInformation = "interview:session_information:"
	RedisPrefixIssueCategories             = "issue:categories"

	MaxAttemptVerifyEmail = 3
	RedisTTLDefault       = 1 * time.Hour

	RedisTTLInterviewSegmentMapping     = 5 * time.Minute
	RedisTTLInterviewLastMessage        = 5 * time.Minute
	RedisTTLInterviewTurn               = 10 * time.Minute
	RedisTTLInterviewStartEndTime       = 1 * time.Hour
	RedisTTLEvaluationRubric            = 1 * time.Hour
	RedisTTLInterviewSessionInformation = 1 * time.Hour
	RedisTTLIssueCategories             = 1 * time.Hour
)
