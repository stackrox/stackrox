package check433

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
)

const (
	standardID = "NIST_800_190:4_3_3"
)

func init() {
	framework.MustRegisterNewCheck(
		standardID,
		framework.DeploymentKind,
		[]string{"NetworkGraph", "NetworkPolicies"},
		common.ClusterHasNetworkPolicies)
}
