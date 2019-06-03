package multipliers

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestComponentCountScore(t *testing.T) {
	countMultiplier := NewComponentCount()

	// We need 14 components added for the count tobe 15 (the two already in the deployment are duplicates of each other)
	deployment := getMockDeployment()

	images := getMockImages()
	components := images[0].Scan.Components
	for i := 0; i < 14; i++ {
		components = append(components, &storage.ImageScanComponent{
			Name:    string(i),
			Version: "1.0",
		})
	}
	images[0].Scan.Components = components

	expectedScore := &storage.Risk_Result{
		Name: ComponentCountHeading,
		Factors: []*storage.Risk_Result_Factor{
			{Message: "Image docker.io/library/nginx:1.10 contains 15 components"},
		},
		Score: 1.25,
	}
	score := countMultiplier.Score(deployment, images)
	assert.Equal(t, expectedScore, score)
}
