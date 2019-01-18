package check411

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/generated/storage"
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
		a := common.NewAnder(common.IsPolicyEnabled(p), common.IsPolicyEnforced(p), doesPolicyHaveCVSS(p))

		if a.Execute() {
			framework.Passf(ctx, "Policy '%s' enabled and enforced", p.GetName())
			return
		}
	}

	framework.Fail(ctx, "Policy that disallows images, with a CVSS score above a threshold, to be deployed not found")
}

func doesPolicyHaveCVSS(p *storage.Policy) common.Andable {
	return func() bool {
		return p.GetFields() != nil && p.GetFields().GetCvss() != nil
	}
}
