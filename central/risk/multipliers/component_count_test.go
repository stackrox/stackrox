package multipliers

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/risk"
	"github.com/stretchr/testify/assert"
)

func TestComponentCountScore(t *testing.T) {
	countMultiplier := NewComponentCount()

	// We need 14 components added for the count to be 15

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
		Name: risk.ImageComponentCount.DisplayTitle,
		Factors: []*storage.Risk_Result_Factor{
			{Message: "Image docker.io/library/nginx:1.10 contains 15 components"},
		},
		Score: 1.25,
	}
	score := countMultiplier.Score(context.Background(), images[0])
	assert.Equal(t, expectedScore, score)
}
