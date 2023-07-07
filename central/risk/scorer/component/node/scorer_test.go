package node

import (
	"context"
	"testing"

	nodeComponentMultiplier "github.com/stackrox/rox/central/risk/multipliers/component/node"
	"github.com/stackrox/rox/central/risk/scorer"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/scancomponent"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestScore(t *testing.T) {
	ctx := context.Background()

	mockCtrl := gomock.NewController(t)

	nodeComponent := scorer.GetMockNode().GetScan().GetComponents()[0]
	nodeComponent.GetVulns()[0].Severity = storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY
	nodeComponent.GetVulns()[1].ScoreVersion = storage.EmbeddedVulnerability_V3
	nodeComponent.GetVulns()[1].Severity = storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY
	nodeComponent.GetVulnerabilities()[0].Severity = storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY
	nodeComponent.GetVulnerabilities()[1].CveBaseInfo.ScoreVersion = storage.CVEInfo_V3
	nodeComponent.GetVulnerabilities()[1].Severity = storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY
	nodeScorer := NewNodeComponentScorer()

	// Without user defined function
	expectedRiskScore := 1.28275
	expectedRiskResults := []*storage.Risk_Result{
		{
			Name: nodeComponentMultiplier.VulnerabilitiesHeading,
			Factors: []*storage.Risk_Result_Factor{
				{Message: "Node Component ComponentX version v1 contains 2 CVEs with severities ranging between Low and Critical"},
			},
			Score: 1.28275,
		},
	}

	actualRisk := nodeScorer.Score(ctx, scancomponent.NewFromNodeComponent(nodeComponent), "")
	assert.Equal(t, expectedRiskResults, actualRisk.GetResults())
	assert.InDelta(t, expectedRiskScore, actualRisk.GetScore(), 0.0001)

	mockCtrl.Finish()
}
