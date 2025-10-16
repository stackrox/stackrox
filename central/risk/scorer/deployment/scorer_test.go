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
	"github.com/stackrox/rox/pkg/protoassert"
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
			storage.ListAlert_builder{
				Deployment: &storage.ListAlertDeployment{},
				Policy: storage.ListAlertPolicy_builder{
					Name:     "Test",
					Severity: storage.Severity_CRITICAL_SEVERITY,
				}.Build(),
			}.Build(),
		},
	}, mockEvaluator)

	mockEvaluator.EXPECT().EvaluateBaselinesAndPersistResult(deployment).MaxTimes(2).Return(nil, nil)

	// Without user defined function
	expectedRiskScore := 12.1794405
	expectedRiskResults := []*storage.Risk_Result{
		storage.Risk_Result_builder{
			Name:    deploymentMultiplier.PolicyViolationsHeading,
			Factors: []*storage.Risk_Result_Factor{storage.Risk_Result_Factor_builder{Message: "Test (severity: Critical)"}.Build()},
			Score:   1.96,
		}.Build(),
		storage.Risk_Result_builder{
			Name: imageMultiplier.VulnerabilitiesHeading,
			Factors: []*storage.Risk_Result_Factor{
				storage.Risk_Result_Factor_builder{Message: "Image \"docker.io/library/nginx:1.10\" contains 3 CVEs with severities ranging between Moderate and Critical"}.Build(),
			},
			Score: 1.5535,
		}.Build(),
		storage.Risk_Result_builder{
			Name: deploymentMultiplier.ServiceConfigHeading,
			Factors: []*storage.Risk_Result_Factor{
				storage.Risk_Result_Factor_builder{Message: "Volumes rw volume were mounted RW"}.Build(),
				storage.Risk_Result_Factor_builder{Message: "Secrets secret are used inside the deployment"}.Build(),
				storage.Risk_Result_Factor_builder{Message: "Capabilities ALL were added"}.Build(),
				storage.Risk_Result_Factor_builder{Message: "No capabilities were dropped"}.Build(),
				storage.Risk_Result_Factor_builder{Message: fmt.Sprintf("Container %q in the deployment is privileged", deployment.GetContainers()[0].GetName())}.Build(),
			},
			Score: 2.0,
		}.Build(),
		storage.Risk_Result_builder{
			Name: deploymentMultiplier.ReachabilityHeading,
			Factors: []*storage.Risk_Result_Factor{
				storage.Risk_Result_Factor_builder{Message: "Port 22 is exposed to external clients"}.Build(),
				storage.Risk_Result_Factor_builder{Message: "Port 23 is exposed in the cluster"}.Build(),
				storage.Risk_Result_Factor_builder{Message: "Port 24 is exposed on node interfaces"}.Build(),
			},
			Score: 1.6,
		}.Build(),
		storage.Risk_Result_builder{
			Name: imageMultiplier.ImageAgeHeading,
			Factors: []*storage.Risk_Result_Factor{
				storage.Risk_Result_Factor_builder{Message: "Image \"docker.io/library/nginx:1.10\" is 180 days old"}.Build(),
			},
			Score: 1.25,
		}.Build(),
	}

	actualRisk := scorer.Score(ctx, deployment, getMockImagesRisk())
	protoassert.SlicesEqual(t, expectedRiskResults, actualRisk.GetResults())
	assert.InDelta(t, expectedRiskScore, actualRisk.GetScore(), 0.0001)

	expectedRiskScore = 12.1794405
	actualRisk = scorer.Score(ctx, deployment, getMockImagesRisk())
	protoassert.SlicesEqual(t, expectedRiskResults, actualRisk.GetResults())
	assert.InDelta(t, expectedRiskScore, actualRisk.GetScore(), 0.0001)

	mockCtrl.Finish()
}
