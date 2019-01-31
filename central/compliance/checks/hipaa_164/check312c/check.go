package check312c

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
)

const checkID = "HIPAA_164:312_c"

func init() {
	framework.MustRegisterNewCheck(
		checkID,
		framework.DeploymentKind,
		[]string{"NetworkGraph", "NetworkPolicies"},
		common.ClusterHasNetworkPolicies)
}
