package constants

import "time"

const (
	RubricNameGeneralInterview = "General Rubric"

	MaxRetryDbEvaluationTx   = 3
	RetryDelayDbEvaluationTx = 200 * time.Millisecond

	PercentageLowThreshold    = 20.0
	PercentageMediumThreshold = 30.0
	PercentageHighThreshold   = 40.0

	PercentageColorLow    = "#DC3545"
	PercentageColorMedium = "#FFC107"
	PercentageColorHigh   = "#28A745"
)
