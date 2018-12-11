package multipliers

import (
	"fmt"

	"github.com/stackrox/rox/generated/storage"
)

const (
	// ReachabilityHeading is the risk result name for scores calculated by this multiplier.
	ReachabilityHeading = `Service Reachability`

	reachabilitySaturation = 10
	reachabilityValue      = 2
)

// reachabilityMultiplier is a scorer for the port exposures
type reachabilityMultiplier struct{}

// NewReachability provides a multiplier that scores the data based on the port exposure
// configuration in the deployment
func NewReachability() Multiplier {
	return &reachabilityMultiplier{}
}

// Score takes a deployment and evaluates its risk based on the service configuration
func (s *reachabilityMultiplier) Score(deployment *storage.Deployment) *storage.Risk_Result {
	var score float32
	riskResult := &storage.Risk_Result{
		Name: ReachabilityHeading,
	}
	for _, c := range deployment.GetContainers() {
		for _, p := range c.GetPorts() {
			score += exposureValue(p.GetExposure())

			riskResult.Factors = append(riskResult.Factors,
				&storage.Risk_Result_Factor{Message: fmt.Sprintf("Container %s exposes port %d %s",
					c.GetImage().GetName().GetRemote(), p.GetExposedPort(), exposureString(p.GetExposure()))})
		}
	}
	if score == 0 {
		return nil
	}
	riskResult.Score = normalizeScore(score, reachabilitySaturation, reachabilityValue)
	return riskResult
}

func exposureValue(exposure storage.PortConfig_Exposure) float32 {
	switch exposure {
	case storage.PortConfig_EXTERNAL:
		return 3
	case storage.PortConfig_NODE:
		return 2
	case storage.PortConfig_INTERNAL:
		return 1
	default:
		return 0
	}
}

func exposureString(exposure storage.PortConfig_Exposure) string {
	switch exposure {
	case storage.PortConfig_INTERNAL:
		return "in the cluster"
	case storage.PortConfig_NODE:
		return "on node interfaces"
	case storage.PortConfig_EXTERNAL:
		return "to external clients"
	default:
		return "in an unknown manner"
	}
}
