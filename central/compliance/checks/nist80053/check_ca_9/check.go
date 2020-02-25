package checkca9

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/pkg/features"
)

const (
	controlID = "NIST_SP_800_53:CA_9"

	interpretationText = `StackRox enables automated container-level network segmentation, preventing data access
through unrestricted network connections. Therefore, the cluster is compliant if all the deployments have ingress
and egress network policies.`
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
