package constants

// Queue Constants
var (
	TaskDeleteFile                    = "task:delete_file"
	TaskSetRedis                      = "task:set_redis"
	TaskDeleteRedis                   = "task:delete_redis"
	TaskDeleteJobRequirement          = "task:delete_job_requirement"
	TaskCalculateTurnScore            = "task:calculate_turn_score"
	TaskCalculateEvaluationInOldState = "task:calculate_evaluation_in_old_state"
	TaskEndInterviewSession           = "task:end_interview_session"

	QueueAiAgent                  = "ai_agent"
	QueueCritical                 = "critical"
	QueueDefault                  = "default"
	MaxRetry                      = 3
	MaxRetryEndInterviewSession   = 7
	RetryDelayEndInterviewSession = 5

	CriticalQueueConcurrency = 10
	DefaultQueueConcurrency  = 5
	DefaultConcurrency       = 10
)
