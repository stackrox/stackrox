package multipliers

import (
	"testing"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stretchr/testify/assert"
)

func TestImageAgeScore(t *testing.T) {
	portMultiplier := NewImageAge()

	deployment := getMockDeployment()
	expectedScore := &v1.Risk_Result{
		Name: ImageAgeHeading,
		Factors: []*v1.Risk_Result_Factor{
			{Message: "Deployment contains an image 180 days old"},
		},
		Score: 1.25,
	}
	score := portMultiplier.Score(deployment)
	assert.Equal(t, expectedScore, score)
}
