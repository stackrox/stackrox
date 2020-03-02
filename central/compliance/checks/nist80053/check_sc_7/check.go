package checksc7

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/pkg/features"
)

const (
	controlID = "NIST_SP_800_53:SC_7"

	interpretationText = common.CheckNetworkPoliciesByDeploymentInterpretation
)

func init() {
	framework.MustRegisterNewCheckIfFlagEnabled(
		framework.CheckMetadata{
			ID:                 controlID,
			Scope:              framework.DeploymentKind,
			DataDependencies:   []string{"NetworkGraph", "NetworkPolicies"},
			InterpretationText: interpretationText,
		},
		func(ctx framework.ComplianceContext) {
			common.CheckNetworkPoliciesByDeployment(ctx)
		}, features.NistSP800_53)
}
