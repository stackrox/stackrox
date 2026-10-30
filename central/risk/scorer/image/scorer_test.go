package image

import (
	"context"
	"testing"

	imageMultiplier "github.com/stackrox/rox/central/risk/multipliers/image"
	pkgScorer "github.com/stackrox/rox/central/risk/scorer"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestScore(t *testing.T) {
	ctx := context.Background()

	mockCtrl := gomock.NewController(t)

	image := pkgScorer.GetMockImage()
	scorer := NewImageScorer()

	// Without user defined function
	expectedRiskScore := 2.525
	expectedRiskResults := []*storage.Risk_Result{
		{
			Name: imageMultiplier.VulnerabilitiesHeading,
			Factors: []*storage.Risk_Result_Factor{
				{Message: "Image \"docker.io/library/nginx:1.10\" contains 3 distinct CVEs with severities ranging between Moderate and Critical"},
			},
			Score: 2.02,
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
	protoassert.SlicesEqual(t, expectedRiskResults, actualRisk.GetResults())
	assert.InDelta(t, expectedRiskScore, actualRisk.GetScore(), 0.0001)
	assert.Equal(t, &storage.RiskVulnerabilitySummary{
		MaxSeverity:    storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
		CriticalCount:  1,
		ImportantCount: 1,
		ModerateCount:  1,
	}, actualRisk.GetVulnerabilitySummary())

	actualRisk = scorer.Score(ctx, image)
	protoassert.SlicesEqual(t, expectedRiskResults, actualRisk.GetResults())
	assert.InDelta(t, expectedRiskScore, actualRisk.GetScore(), 0.0001)

	mockCtrl.Finish()
}

func TestScoreV2(t *testing.T) {
	testutils.MustUpdateFeature(t, features.FlattenImageData, true)
	ctx := context.Background()

	mockCtrl := gomock.NewController(t)

	image := pkgScorer.GetMockImageV2()
	scorer := NewImageScorer()

	// Without user defined function
	expectedRiskScore := 2.525
	expectedRiskResults := []*storage.Risk_Result{
		{
			Name: imageMultiplier.VulnerabilitiesHeading,
			Factors: []*storage.Risk_Result_Factor{
				{Message: "Image \"docker.io/library/nginx:1.10\" contains 3 distinct CVEs with severities ranging between Moderate and Critical"},
			},
			Score: 2.02,
		},
		{
			Name: imageMultiplier.ImageAgeHeading,
			Factors: []*storage.Risk_Result_Factor{
				{Message: "Image \"docker.io/library/nginx:1.10\" is 180 days old"},
			},
			Score: 1.25,
		},
	}

	actualRisk := scorer.ScoreV2(ctx, image)
	protoassert.SlicesEqual(t, expectedRiskResults, actualRisk.GetResults())
	assert.InDelta(t, expectedRiskScore, actualRisk.GetScore(), 0.0001)
	assert.Equal(t, &storage.RiskVulnerabilitySummary{
		MaxSeverity:    storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
		CriticalCount:  1,
		ImportantCount: 1,
		ModerateCount:  1,
	}, actualRisk.GetVulnerabilitySummary())

	actualRisk = scorer.ScoreV2(ctx, image)
	protoassert.SlicesEqual(t, expectedRiskResults, actualRisk.GetResults())
	assert.InDelta(t, expectedRiskScore, actualRisk.GetScore(), 0.0001)

	mockCtrl.Finish()
}
