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
	common.IsImageScannerInUse(ctx)
	common.CheckBuildTimePolicyEnforced(ctx)
}

func checkCVSS7PolicyEnforced(ctx framework.ComplianceContext) {
	policies := ctx.Data().Policies()
	for _, p := range policies {
		if !policyHasCVSS(p) {
			continue
		}

		enabled := common.IsPolicyEnabled(p)
		enforced := common.IsPolicyEnforced(p)

		if enabled && !enforced {
			framework.Failf(ctx, "Enforcement is not set on the policy that disallows images with a critical CVSS score (%q)", p.GetName())
			return
		}

		if enabled && enforced {
			framework.Passf(ctx, "Policy that disallows images with a critical CVSS score (%q) is enabled and enforced", p.GetName())
			return
		}
	}
	framework.Fail(ctx, "No policy that disallows images with a critical CVSS score was found")
}

func policyHasCVSS(p *storage.Policy) bool {
	return p.GetFields().GetCvss() != nil
}
