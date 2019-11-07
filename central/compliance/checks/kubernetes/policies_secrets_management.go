package kubernetes

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
)

func init() {
	framework.MustRegisterChecks(
		common.PerNodeNoteCheck("CIS_Kubernetes_v1_5:5_4_1", "Prefer using secrets as files over secrets as environment variables"),
		common.PerNodeNoteCheck("CIS_Kubernetes_v1_5:5_4_2", "Consider external secret storage"),
	)
}
