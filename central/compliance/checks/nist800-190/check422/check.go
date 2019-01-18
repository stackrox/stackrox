package check422

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/generated/storage"
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
		a := common.NewAnder(common.IsPolicyEnabled(p), common.IsPolicyEnforced(p), doesPolicyHaveImageTagLatest(p))

		if a.Execute() {
			framework.Passf(ctx, "Policy '%s' enabled and enforced", p.GetName())
			return
		}
	}

	framework.Fail(ctx, "Policy that disallows old images to be deployed not found")
}

func checkLatestImageTagPolicyEnforced(ctx framework.ComplianceContext) {
	policies := ctx.Data().Policies()
	for _, p := range policies {
		a := common.NewAnder(common.IsPolicyEnabled(p), common.IsPolicyEnforced(p), doesPolicyHaveImageAgeDays(p))

		if a.Execute() {
			framework.Passf(ctx, "Policy '%s' enabled and enforced", p.GetName())
			return
		}
	}

	framework.Fail(ctx, "Policy that disallows images with tag 'latest' to be deployed not found")
}

func doesPolicyHaveImageTagLatest(p *storage.Policy) common.Andable {
	return func() bool {
		return p.GetFields() != nil && p.GetFields().GetImageName().GetTag() != "latest"
	}
}

func doesPolicyHaveImageAgeDays(p *storage.Policy) common.Andable {
	return func() bool {
		return p.GetFields() != nil && p.GetFields().GetImageAgeDays() != 0 && !p.GetDisabled()
	}
}
