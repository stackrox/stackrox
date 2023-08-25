package image

import (
	"context"
	"testing"

	imageMultiplier "github.com/stackrox/rox/central/risk/multipliers/image"
	pkgScorer "github.com/stackrox/rox/central/risk/scorer"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestScore(t *testing.T) {
	ctx := context.Background()

	mockCtrl := gomock.NewController(t)

	image := pkgScorer.GetMockImage()
	scorer := NewImageScorer()

	// Without user defined function
	expectedRiskScore := 1.9418751
	expectedRiskResults := []*storage.Risk_Result{
		{
			Name: imageMultiplier.VulnerabilitiesHeading,
			Factors: []*storage.Risk_Result_Factor{
				{Message: "Image \"docker.io/library/nginx:1.10\" contains 3 CVEs with severities ranging between Moderate and Critical"},
			},
			Score: 1.5535,
		},
		{
			Name: imageMultiplier.ImageAgeHeading,
			Factors: []*storage.Risk_Result_Factor{
				{Message: "Image \"docker.io/library/nginx:1.10\" is 180 days old"},
			},
			Score: 1.25,
		},
	}

	actualRisk := scorer.Score(ctx, image)
	assert.Equal(t, expectedRiskResults, actualRisk.GetResults())
	assert.InDelta(t, expectedRiskScore, actualRisk.GetScore(), 0.0001)

	actualRisk = scorer.Score(ctx, image)
	assert.Equal(t, expectedRiskResults, actualRisk.GetResults())
	assert.InDelta(t, expectedRiskScore, actualRisk.GetScore(), 0.0001)

	mockCtrl.Finish()
}
