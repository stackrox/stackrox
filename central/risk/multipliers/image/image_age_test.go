package image

import (
	"context"
	"fmt"
	"testing"

	"github.com/stackrox/rox/central/risk/multipliers"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestImageAgeScore(t *testing.T) {
	imageAgeMultiplier := NewImageAge()

	image := multipliers.GetMockImages()[0]
	expectedScore := &storage.Risk_Result{
		Name: ImageAgeHeading,
		Factors: []*storage.Risk_Result_Factor{
			{Message: fmt.Sprintf("Image %q is 180 days old", image.GetName().GetFullName())},
		},
		Score: 1.25,
	}

	score := imageAgeMultiplier.Score(context.Background(), image)
	assert.Equal(t, expectedScore, score)
}
