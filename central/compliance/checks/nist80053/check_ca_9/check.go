package checkca9

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
)

const (
	controlID = "NIST_SP_800_53_Rev_4:CA_9"

	interpretationText = common.CheckNetworkPoliciesByDeploymentInterpretation
)

func init() {
	framework.MustRegisterNewCheck(
		framework.CheckMetadata{
			ID:                 controlID,
			Scope:              framework.DeploymentKind,
			DataDependencies:   []string{"NetworkGraph", "NetworkPolicies"},
			InterpretationText: interpretationText,
		},
		func(ctx framework.ComplianceContext) {
			common.CheckNetworkPoliciesByDeployment(ctx)
		})
}
