package check411

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
)

const (
	standardID = "NIST_800_190:4_1_1"
)

func init() {
	framework.MustRegisterNewCheck(
		standardID,
		framework.ClusterKind,
		[]string{"Policies", "ImageIntegrations"},
		func(ctx framework.ComplianceContext) {
			checkNIST411(ctx)
		})
}

func checkNIST411(ctx framework.ComplianceContext) {
	checkCVSS7PolicyEnforced(ctx)
	common.CheckImageScannerInUse(ctx)
	common.CheckBuildTimePolicyEnforced(ctx)
}

func checkCVSS7PolicyEnforced(ctx framework.ComplianceContext) {
	common.CheckPolicyEnforced(ctx, "CVSS >= 7")
}
