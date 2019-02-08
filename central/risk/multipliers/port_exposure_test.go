package multipliers

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestPortExposureScore(t *testing.T) {
	portMultiplier := NewReachability()

	deployment := getMockDeployment()
	expectedScore := &storage.Risk_Result{
		Name: ReachabilityHeading,
		Factors: []*storage.Risk_Result_Factor{
			{Message: "Port 8082 is exposed to external clients"},
			{Message: "Port 8083 is exposed in the cluster"},
			{Message: "Port 8084 is exposed on node interfaces"},
		},
		Score: 1.6,
	}
	score := portMultiplier.Score(deployment)
	assert.Equal(t, expectedScore, score)
}
