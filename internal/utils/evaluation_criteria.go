package utils

import (
	"strconv"

	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
)

func ConvertWeightToPercentage(weight string) string {
	numericWeight, err := strconv.ParseFloat(weight, 64)
	if err != nil {
		return "0.0"
	}

	percentage := numericWeight * 100
	rounded := float64(int(percentage*10+0.5)) / 10
	return strconv.FormatFloat(rounded, 'f', 1, 64)
}

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
