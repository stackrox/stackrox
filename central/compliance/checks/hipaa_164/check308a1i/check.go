package check308a1i

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
)

const checkID = "HIPAA_164:308_a_1_i"

func init() {
	framework.MustRegisterNewCheck(
		checkID,
		framework.ClusterKind,
		[]string{"ImageIntegrations"},
		clusterIsCompliant)
}

func clusterIsCompliant(ctx framework.ComplianceContext) {
	common.IsImageScannerInUse(ctx)
	common.CheckImageScannerWasUsed(ctx)
}
