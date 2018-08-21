package multipliers

import (
	"testing"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stretchr/testify/assert"
)

func TestPortExposureScore(t *testing.T) {
	portMultiplier := NewReachability()

	deployment := getMockDeployment()
	expectedScore := &v1.Risk_Result{
		Name: ReachabilityHeading,
		Factors: []*v1.Risk_Result_Factor{
			{Message: "Container library/nginx exposes port 8082 to external clients"},
			{Message: "Container library/nginx exposes port 8083 in the cluster"},
			{Message: "Container library/nginx exposes port 8084 on node interfaces"},
		},
		Score: 1.6,
	}
	score := portMultiplier.Score(deployment)
	assert.Equal(t, expectedScore, score)
}
