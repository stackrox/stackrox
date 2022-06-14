package check132

import (
	"github.com/stackrox/stackrox/central/compliance/checks/common"
	"github.com/stackrox/stackrox/central/compliance/framework"
	pkgFramework "github.com/stackrox/stackrox/pkg/compliance/framework"
)

const checkID = "PCI_DSS_3_2:1_3_2"

func init() {
	framework.MustRegisterNewCheck(
		framework.CheckMetadata{
			ID:                 checkID,
			Scope:              pkgFramework.DeploymentKind,
			DataDependencies:   []string{"NetworkGraph", "NetworkPolicies"},
			InterpretationText: interpretationText,
		},
		common.ClusterHasIngressNetworkPolicies)
}
