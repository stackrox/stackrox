package risk

import (
	"testing"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/stretchr/testify/assert"
)

func TestPortExposureScore(t *testing.T) {
	portMultiplier := newReachabilityMultiplier()

	deployment := getMockDeployment()
	expectedScore := &v1.Risk_Result{
		Name: reachabilityHeading,
		Factors: []string{
			"Container library/nginx exposes port 22 to external clients",
			"Container library/nginx exposes port 23 in the cluster",
			"Container library/nginx exposes port 8080 on node interfaces",
		},
		Score: 1.6,
	}
	score := portMultiplier.Score(deployment)
	assert.Equal(t, expectedScore, score)
}
