package deployment

import (
	"context"
	"fmt"
	"testing"

	evaluatorMocks "github.com/stackrox/rox/central/processbaseline/evaluator/mocks"
	"github.com/stackrox/rox/central/risk/getters"
	deploymentMultiplier "github.com/stackrox/rox/central/risk/multipliers/deployment"
	imageMultiplier "github.com/stackrox/rox/central/risk/multipliers/image"
	pkgScorer "github.com/stackrox/rox/central/risk/scorer"
	"github.com/stackrox/rox/central/risk/scorer/image"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

// GetMockImagesRisk returns a slice of mock image risk
func getMockImagesRisk() []*storage.Risk {
	return []*storage.Risk{
		GetMockImageRisk(),
	}
}

// GetMockImageRisk returns the risk for the mock image
func GetMockImageRisk() *storage.Risk {
	scorer := image.NewImageScorer()
	return scorer.Score(context.Background(), pkgScorer.GetMockImage())
}

func TestScore(t *testing.T) {
	ctx := context.Background()

	mockCtrl := gomock.NewController(t)
	mockEvaluator := evaluatorMocks.NewMockEvaluator(mockCtrl)

	deployment := pkgScorer.GetMockDeployment()
	scorer := NewDeploymentScorer(&getters.MockAlertsSearcher{
		Alerts: []*storage.ListAlert{
			{
				Entity: &storage.ListAlert_Deployment{Deployment: &storage.ListAlertDeployment{}},
				Policy: &storage.ListAlertPolicy{
					Name:     "Test",
					Severity: storage.Severity_CRITICAL_SEVERITY,
				},
			},
		},
	}, mockEvaluator)

	mockEvaluator.EXPECT().EvaluateBaselinesAndPersistResult(deployment).MaxTimes(2).Return(nil, nil)

	// Without user defined function
	expectedRiskScore := 12.1794405
	expectedRiskResults := []*storage.Risk_Result{
		{
			Name:    deploymentMultiplier.PolicyViolationsHeading,
			Factors: []*storage.Risk_Result_Factor{{Message: "Test (severity: Critical)"}},
			Score:   1.96,
		},
		{
			Name: imageMultiplier.VulnerabilitiesHeading,
			Factors: []*storage.Risk_Result_Factor{
				{Message: "Image \"docker.io/library/nginx:1.10\" contains 3 CVEs with severities ranging between Moderate and Critical"},
			},
			Score: 1.5535,
		},
		{
			Name: deploymentMultiplier.ServiceConfigHeading,
			Factors: []*storage.Risk_Result_Factor{
				{Message: "Volumes rw volume were mounted RW"},
				{Message: "Secrets secret are used inside the deployment"},
				{Message: "Capabilities ALL were added"},
				{Message: "No capabilities were dropped"},
				{Message: fmt.Sprintf("Container %q in the deployment is privileged", deployment.GetContainers()[0].GetName())},
			},
			Score: 2.0,
		},
		{
			Name: deploymentMultiplier.ReachabilityHeading,
			Factors: []*storage.Risk_Result_Factor{
				{Message: "Port 22 is exposed to external clients"},
				{Message: "Port 23 is exposed in the cluster"},
				{Message: "Port 24 is exposed on node interfaces"},
			},
			Score: 1.6,
		},
		{
			Name: imageMultiplier.ImageAgeHeading,
			Factors: []*storage.Risk_Result_Factor{
				{Message: "Image \"docker.io/library/nginx:1.10\" is 180 days old"},
			},
			Score: 1.25,
		},
	}

	actualRisk := scorer.Score(ctx, deployment, getMockImagesRisk())
	assert.Equal(t, expectedRiskResults, actualRisk.GetResults())
	assert.InDelta(t, expectedRiskScore, actualRisk.GetScore(), 0.0001)

	expectedRiskScore = 12.1794405
	actualRisk = scorer.Score(ctx, deployment, getMockImagesRisk())
	assert.Equal(t, expectedRiskResults, actualRisk.GetResults())
	assert.InDelta(t, expectedRiskScore, actualRisk.GetScore(), 0.0001)

	mockCtrl.Finish()
}
