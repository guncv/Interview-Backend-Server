package constants

import "time"

const (
	RubricNameGeneralInterview = "General Rubric"

	MaxRetryDbEvaluationTx   = 3
	RetryDelayDbEvaluationTx = 200 * time.Millisecond
)
