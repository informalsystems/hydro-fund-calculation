package process

import (
	"strings"

	"fund_calculation/config"
)

func venueFraction(deploymentType string) float64 {
	return config.GlobalConfig.VenueFractions[strings.ToLower(deploymentType)]
}
