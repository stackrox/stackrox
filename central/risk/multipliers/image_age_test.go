package multipliers

import (
	"context"
	"fmt"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/risk"
	"github.com/stretchr/testify/assert"
)

func TestImageAgeScore(t *testing.T) {
	imageAgeMultiplier := NewImageAge()

	image := getMockImages()[0]
	expectedScore := &storage.Risk_Result{
		Name: risk.ImageAge.DisplayTitle,
		Factors: []*storage.Risk_Result_Factor{
			{Message: fmt.Sprintf("Image %q is 180 days old", image.GetName().GetFullName())},
		},
		Score: 1.25,
	}

	score := imageAgeMultiplier.Score(context.Background(), image)
	assert.Equal(t, expectedScore, score)
}
