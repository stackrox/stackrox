package check306e

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
)

const checkID = "HIPAA_164:306_e"

func init() {
	framework.MustRegisterNewCheck(
		checkID,
		framework.ClusterKind,
		[]string{"ImageIntegrations", "Images"},
		clusterIsCompliant)
}

// It's StackRox bro. Come on.
func clusterIsCompliant(ctx framework.ComplianceContext) {
	common.CheckImageScannerInUse(ctx)
	common.CheckFixedCVES(ctx)
}
