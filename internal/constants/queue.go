package constants

// Queue Constants
var (
	TaskSendResetPasswordEmail = "task:send_reset_password_email"
	TaskSendVerifyEmail        = "task:send_verify_email"
	TaskDeleteFile             = "task:delete_file"
	TaskSetRedis               = "task:set_redis"
	TaskDeleteRedis            = "task:delete_redis"
	TaskDeleteJobRequirement   = "task:delete_job_requirement"
	TaskCalculateTurnScore     = "task:calculate_turn_score"

	QueueCritical = "critical"
	QueueDefault  = "default"
	MaxRetry      = 3

	CriticalQueueConcurrency = 10
	DefaultQueueConcurrency  = 5
	DefaultConcurrency       = 10
)
