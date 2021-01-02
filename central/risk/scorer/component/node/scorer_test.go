package node

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	nodeComponentMultiplier "github.com/stackrox/rox/central/risk/multipliers/component/node"
	"github.com/stackrox/rox/central/risk/scorer"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestScore(t *testing.T) {
	ctx := context.Background()

	mockCtrl := gomock.NewController(t)

	imageComponent := scorer.GetMockNode().GetScan().GetComponents()[0]
	nodeScorer := NewNodeComponentScorer()

	// Without user defined function
	expectedRiskScore := 1.15
	expectedRiskResults := []*storage.Risk_Result{
		{
			Name: nodeComponentMultiplier.VulnerabilitiesHeading,
			Factors: []*storage.Risk_Result_Factor{
				{Message: "Node Component ComponentX version v1 contains 2 CVEs with CVSS scores ranging between 5.0 and 5.0"},
			},
			Score: 1.15,
		},
	}

	actualRisk := nodeScorer.Score(ctx, imageComponent)
	assert.Equal(t, expectedRiskResults, actualRisk.GetResults())
	assert.InDelta(t, expectedRiskScore, actualRisk.GetScore(), 0.0001)

	mockCtrl.Finish()
}
