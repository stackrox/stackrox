package multipliers

import (
	"fmt"

	"github.com/stackrox/rox/generated/api/v1"
)

const (
	// ReachabilityHeading is the risk result name for scores calculated by this multiplier.
	ReachabilityHeading = `Service Reachability`

	reachabilitySaturation = float32(10)
)

// reachabilityMultiplier is a scorer for the port exposures
type reachabilityMultiplier struct{}

// NewReachability provides a multiplier that scores the data based on the port exposure
// configuration in the deployment
func NewReachability() Multiplier {
	return &reachabilityMultiplier{}
}

// Score takes a deployment and evaluates its risk based on the service configuration
func (s *reachabilityMultiplier) Score(deployment *v1.Deployment) *v1.Risk_Result {
	var score float32
	riskResult := &v1.Risk_Result{
		Name: ReachabilityHeading,
	}
	for _, c := range deployment.GetContainers() {
		for _, p := range c.GetPorts() {
			score += exposureValue(p.GetExposure())

			riskResult.Factors = append(riskResult.Factors, fmt.Sprintf("Container %s exposes port %d %s",
				c.GetImage().GetName().GetRemote(), p.GetExposedPort(), exposureString(p.GetExposure())))
		}
	}
	if score == 0 {
		return nil
	}
	if score > reachabilitySaturation {
		score = reachabilitySaturation
	}
	riskResult.Score = (score / reachabilitySaturation) + 1
	return riskResult
}

func exposureValue(exposure v1.PortConfig_Exposure) float32 {
	switch exposure {
	case v1.PortConfig_EXTERNAL:
		return 3
	case v1.PortConfig_NODE:
		return 2
	case v1.PortConfig_INTERNAL:
		return 1
	default:
		return 0
	}
}

func exposureString(exposure v1.PortConfig_Exposure) string {
	switch exposure {
	case v1.PortConfig_INTERNAL:
		return "in the cluster"
	case v1.PortConfig_NODE:
		return "on node interfaces"
	case v1.PortConfig_EXTERNAL:
		return "to external clients"
	default:
		return "in an unknown manner"
	}
}
