package kubernetes

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
)

func init() {
	framework.MustRegisterChecks(
		common.PerNodeNoteCheck("CIS_Kubernetes_v1_5:5_3_1", "Ensure that the CNI in use supports Network Policies"),
		// TODO: @boo - implement the check below
		common.PerNodeNoteCheck("CIS_Kubernetes_v1_5:5_3_2", "Ensure that all Namespaces have Network Policies defined"),
	)
}
