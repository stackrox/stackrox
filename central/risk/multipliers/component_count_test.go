package multipliers

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestComponentCountScore(t *testing.T) {
	portMultiplier := NewComponentCount()

	// We need 14 components added for the count tobe 15 (the two already in the deployment are duplicates of each other)
	deployment := getMockDeployment()
	components := deployment.Containers[0].Image.Scan.Components
	for i := 0; i < 14; i++ {
		components = append(components, &storage.ImageScanComponent{
			Name:    string(i),
			Version: "1.0",
		})
	}
	deployment.Containers[0].Image.Scan.Components = components

	expectedScore := &storage.Risk_Result{
		Name: ComponentCountHeading,
		Factors: []*storage.Risk_Result_Factor{
			{Message: "image docker.io/library/nginx:1.10 contains 15 components"},
		},
		Score: 1.25,
	}
	score := portMultiplier.Score(deployment)
	assert.Equal(t, expectedScore, score)
}
