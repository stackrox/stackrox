package multipliers

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestImageAgeScore(t *testing.T) {
	imageAgeMultiplier := NewImageAge()

	deployment := getMockDeployment()
	expectedScore := &storage.Risk_Result{
		Name: ImageAgeHeading,
		Factors: []*storage.Risk_Result_Factor{
			{Message: "Deployment contains an image 180 days old"},
		},
		Score: 1.25,
	}

	score := imageAgeMultiplier.Score(context.Background(), deployment, getMockImages())
	assert.Equal(t, expectedScore, score)
}
