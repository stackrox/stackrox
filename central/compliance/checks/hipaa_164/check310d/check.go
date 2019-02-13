package check310d

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
)

const checkID = "HIPAA_164:310_d"

func init() {
	framework.MustRegisterNewCheck(
		framework.CheckMetadata{
			ID:                 checkID,
			Scope:              framework.ClusterKind,
			InterpretationText: interpretationText,
		},
		common.CheckRuntimeSupportInCluster)
}
