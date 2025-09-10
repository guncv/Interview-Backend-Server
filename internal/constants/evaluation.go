package constants

import "time"

const (
	RubricNameGeneralInterview = "General Interview Rubric (v1)"

	MaxRetryDbEvaluationTx   = 3
	RetryDelayDbEvaluationTx = 200 * time.Millisecond
)
