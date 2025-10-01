package constants

// Queue Constants
var (
	TaskSendResetPasswordEmail        = "task:send_reset_password_email"
	TaskSendVerifyEmail               = "task:send_verify_email"
	TaskDeleteFile                    = "task:delete_file"
	TaskSetRedis                      = "task:set_redis"
	TaskDeleteRedis                   = "task:delete_redis"
	TaskDeleteJobRequirement          = "task:delete_job_requirement"
	TaskCalculateTurnScore            = "task:calculate_turn_score"
	TaskCalculateEvaluationInOldState = "task:calculate_evaluation_in_old_state"
	TaskEndInterviewSession           = "task:end_interview_session"

	QueueCritical = "critical"
	QueueDefault  = "default"
	MaxRetry      = 3

	CriticalQueueConcurrency = 10
	DefaultQueueConcurrency  = 5
	DefaultConcurrency       = 10
)
