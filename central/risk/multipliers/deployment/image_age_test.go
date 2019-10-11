package deployment

import (
	"context"
	"fmt"
	"testing"

	"github.com/stackrox/rox/central/risk/multipliers"
	imageMultiplier "github.com/stackrox/rox/central/risk/multipliers/image"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestImageAgeScore(t *testing.T) {
	imageAgeMultiplier := NewImageAge()

	deployment := multipliers.GetMockDeployment()
	images := multipliers.GetMockImages()
	expectedScore := &storage.Risk_Result{
		Name: imageMultiplier.ImageAgeHeading,
		Factors: []*storage.Risk_Result_Factor{
			{Message: fmt.Sprintf("Image %q is 180 days old", images[0].GetName().GetFullName())},
		},
		Score: 1.25,
	}

	score := imageAgeMultiplier.Score(context.Background(), deployment, images)
	assert.Equal(t, expectedScore, score)
}
