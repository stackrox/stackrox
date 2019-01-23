package check132

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
)

const checkID = "PCI_DSS_3_2:1_3_2"

func init() {
	framework.MustRegisterNewCheck(
		checkID,
		framework.DeploymentKind,
		[]string{"NetworkGraph", "NetworkPolicies"},
		common.ClusterHasIngressNetworkPolicies)
}
