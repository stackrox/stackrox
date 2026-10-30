package deployment

import (
	"context"
	"fmt"

	"github.com/stackrox/rox/central/risk/multipliers"
	"github.com/stackrox/rox/generated/storage"
)

const (
	// ReachabilityHeading is the risk result name for scores calculated by this multiplier.
	ReachabilityHeading = `Service Reachability`

	reachabilitySaturation = 10
	reachabilityValue      = 4

	additionalPortContribution    = 0.2
	additionalPortContributionCap = 2
)

// reachabilityMultiplier is a scorer for the port exposures
type reachabilityMultiplier struct{}

// NewReachability provides a multiplier that scores the data based on the port exposure
// configuration in the deployment
func NewReachability() Multiplier {
	return &reachabilityMultiplier{}
}

// Score takes a deployment and evaluates its risk based on the service configuration
func (s *reachabilityMultiplier) Score(_ context.Context, deployment *storage.Deployment, _ map[string][]*storage.Risk_Result) *storage.Risk_Result {
	var maxExposureScore float32
	riskResult := &storage.Risk_Result{
		Name: ReachabilityHeading,
	}
	for _, p := range deployment.GetPorts() {
		maxExposureScore = max(maxExposureScore, exposureValue(p.GetExposure()))

		riskResult.Factors = append(riskResult.Factors,
			&storage.Risk_Result_Factor{Message: fmt.Sprintf("Port %d is exposed %s",
				p.GetContainerPort(), exposureString(p.GetExposure()))})
	}
	if maxExposureScore == 0 {
		return nil
	}
	score := maxExposureScore
	if additionalPorts := len(deployment.GetPorts()) - 1; additionalPorts > 0 {
		score += min(float32(additionalPorts)*additionalPortContribution, additionalPortContributionCap)
	}
	riskResult.Score = multipliers.NormalizeScore(score, reachabilitySaturation, reachabilityValue)
	return riskResult
}

func exposureValue(exposure storage.PortConfig_ExposureLevel) float32 {
	switch exposure {
	case storage.PortConfig_EXTERNAL, storage.PortConfig_ROUTE:
		return 10
	case storage.PortConfig_NODE, storage.PortConfig_HOST:
		return 5
	case storage.PortConfig_INTERNAL:
		return 1
	default:
		return 0
	}
}

func exposureString(exposure storage.PortConfig_ExposureLevel) string {
	switch exposure {
	case storage.PortConfig_INTERNAL:
		return "in the cluster"
	case storage.PortConfig_HOST:
		return "on every node running the deployment"
	case storage.PortConfig_NODE:
		return "on node interfaces"
	case storage.PortConfig_EXTERNAL, storage.PortConfig_ROUTE:
		return "to external clients"
	default:
		return "in an unknown manner"
	}
}
