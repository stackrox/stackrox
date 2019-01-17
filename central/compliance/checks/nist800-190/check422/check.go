package check422

import (
	"github.com/stackrox/rox/central/compliance/framework"
)

const (
	standardID = "NIST_800_190:4_2_2"
)

func init() {
	framework.MustRegisterNewCheck(
		standardID,
		framework.ClusterKind,
		[]string{"Policies", "ImageIntegrations"},
		func(ctx framework.ComplianceContext) {
			checkNIST422(ctx)
		})
}

func checkNIST422(ctx framework.ComplianceContext) {
	checkLatestImageTagPolicyEnforced(ctx)
	checkImageAgePolicyEnforced(ctx)
}

func checkImageAgePolicyEnforced(ctx framework.ComplianceContext) {
	policies := ctx.Data().Policies()
	for _, p := range policies {
		if p.GetFields() != nil && p.GetFields().GetImageAgeDays() != 0 && !p.GetDisabled() && len(p.GetEnforcementActions()) != 0 {
			framework.Passf(ctx, "Policy '%s' enabled and enforced", p.GetName())
			return
		}
	}

	framework.Fail(ctx, "Policy that disallows old images to be deployed not found")
}

func checkLatestImageTagPolicyEnforced(ctx framework.ComplianceContext) {
	policies := ctx.Data().Policies()
	for _, p := range policies {
		if p.GetFields() != nil && p.GetFields().GetImageName().GetTag() != "latest" && !p.GetDisabled() && len(p.GetEnforcementActions()) != 0 {
			framework.Passf(ctx, "Policy '%s' enabled and enforced", p.GetName())
			return
		}
	}

	framework.Fail(ctx, "Policy that disallows images with tag 'latest' to be deployed not found")
}
