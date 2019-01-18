package check442

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
)

const (
	standardID = "NIST_800_190:4_4_2"
)

func init() {
	framework.MustRegisterNewCheck(
		standardID,
		framework.DeploymentKind,
		[]string{"NetworkGraph", "NetworkPolicies"},
		common.ClusterHasNetworkPolicies)
}
