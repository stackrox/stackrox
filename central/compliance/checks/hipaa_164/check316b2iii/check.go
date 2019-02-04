package check316b2ii

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
)

const checkID = "HIPAA_164:316_b_2_iii"

func init() {
	framework.MustRegisterNewCheck(
		framework.CheckMetadata{
			ID:                 checkID,
			Scope:              framework.ClusterKind,
			DataDependencies:   []string{"ImageIntegrations"},
			InterpretationText: interpretationText,
		},
		common.IsImageScannerInUse)
}
