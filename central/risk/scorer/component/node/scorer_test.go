package node

import (
	"context"
	"testing"

	nodeComponentMultiplier "github.com/stackrox/rox/central/risk/multipliers/component/node"
	"github.com/stackrox/rox/central/risk/scorer"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/scancomponent"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestScore(t *testing.T) {
	ctx := context.Background()

	mockCtrl := gomock.NewController(t)

	nodeComponent := scorer.GetMockNode().GetScan().GetComponents()[0]
	nodeComponent.GetVulns()[0].SetSeverity(storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY)
	nodeComponent.GetVulns()[1].SetScoreVersion(storage.EmbeddedVulnerability_V3)
	nodeComponent.GetVulns()[1].SetSeverity(storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY)
	nodeComponent.GetVulnerabilities()[0].SetSeverity(storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY)
	nodeComponent.GetVulnerabilities()[1].GetCveBaseInfo().SetScoreVersion(storage.CVEInfo_V3)
	nodeComponent.GetVulnerabilities()[1].SetSeverity(storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY)
	nodeScorer := NewNodeComponentScorer()

	// Without user defined function
	expectedRiskScore := 1.28275
	expectedRiskResults := []*storage.Risk_Result{
		storage.Risk_Result_builder{
			Name: nodeComponentMultiplier.VulnerabilitiesHeading,
			Factors: []*storage.Risk_Result_Factor{
				storage.Risk_Result_Factor_builder{Message: "Node Component ComponentX version v1 contains 2 CVEs with severities ranging between Low and Critical"}.Build(),
			},
			Score: 1.28275,
		}.Build(),
	}

	actualRisk := nodeScorer.Score(ctx, scancomponent.NewFromNodeComponent(nodeComponent), "")
	protoassert.SlicesEqual(t, expectedRiskResults, actualRisk.GetResults())
	assert.InDelta(t, expectedRiskScore, actualRisk.GetScore(), 0.0001)

	mockCtrl.Finish()
}
