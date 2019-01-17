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
	policies := ctx.Data().Policies()
	for _, p := range policies {
		if p.GetFields() != nil && p.GetFields().GetCvss() != nil && !p.GetDisabled() && len(p.GetEnforcementActions()) != 0 {
			framework.Passf(ctx, "Policy '%s' enabled and enforced", p.GetName())
			return
		}
	}

	framework.Fail(ctx, "Policy that disallows images, with a CVSS score above a threshold, to be deployed not found")
}
