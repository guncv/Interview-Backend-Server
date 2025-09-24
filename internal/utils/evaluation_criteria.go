package utils

import (
	"strconv"

	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
)

// ConvertWeightToPercentage converts a weight given as a string (expected as a decimal fraction, e.g. "0.25")
// into a percentage string with one decimal place (e.g. "25.0").
// If the input cannot be parsed as a float, it returns "0.0".
// The result is rounded to one decimal place using standard half-up rounding.
func ConvertWeightToPercentage(weight string) string {
	numericWeight, err := strconv.ParseFloat(weight, 64)
	if err != nil {
		return "0.0"
	}

	percentage := numericWeight * 100
	rounded := float64(int(percentage*10+0.5)) / 10
	return strconv.FormatFloat(rounded, 'f', 1, 64)
}

// GetPercentageColor returns a color constant corresponding to the numeric percentage
// provided as a string. The input is parsed as a float64; if parsing fails the low
// percentage color is returned. Thresholds: values <= 20 → PercentageColorLow,
// values <= 30 → PercentageColorMedium, values > 30 → PercentageColorHigh.
func GetPercentageColor(percentage string) string {
	numericPercentage, err := strconv.ParseFloat(percentage, 64)
	if err != nil {
		return constants.PercentageColorLow
	}

	if numericPercentage <= 20 {
		return constants.PercentageColorLow
	} else if numericPercentage <= 30 {
		return constants.PercentageColorMedium
	} else {
		return constants.PercentageColorHigh
	}
}
