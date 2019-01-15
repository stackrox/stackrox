package check411

import (
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/generated/storage"
)

func init() {
	framework.MustRegisterNewCheck(
		"NIST-800-190:4.1.1",
		framework.ClusterKind,
		[]string{"Policies", "ImageIntegrations"},
		func(ctx framework.ComplianceContext) {
			checkNIST411(ctx)
		})
}

func checkNIST411(ctx framework.ComplianceContext) {
	checkCVSS7PolicyEnforced(ctx)
	checkImageScannerInUse(ctx)
	checkBuildTimePolicyEnforced(ctx)
}

func checkCVSS7PolicyEnforced(ctx framework.ComplianceContext) {
	checkPolicyEnforced(ctx, "CVSS >= 7")
}

func checkImageScannerInUse(ctx framework.ComplianceContext) {
	imageIntegrations := ctx.Data().ImageIntegrations()

	if len(imageIntegrations) == 0 {
		framework.Fail(ctx, "No image scanner integrations have been configured")
		return
	}

	framework.Pass(ctx, "At least one image integration has been configured")
}

func checkBuildTimePolicyEnforced(ctx framework.ComplianceContext) {
	policies := ctx.Data().Policies()
	for _, p := range policies {
		for _, stage := range p.GetLifecycleStages() {
			if stage == storage.LifecycleStage_BUILD && !p.Disabled && len(p.EnforcementActions) != 0 {
				framework.Pass(ctx, "At least one build time policy is enabled and enforced")
				return
			}
		}
	}

	framework.Fail(ctx, "Unable to find a build time policy that is enabled and enforced")
}

func checkPolicyEnforced(ctx framework.ComplianceContext, name string) {
	policies := ctx.Data().Policies()
	p := policies[name]

	if p.GetDisabled() {
		framework.Fail(ctx, "Policy 'CVSS >= 7' not enabled")
		return
	}

	if len(p.GetEnforcementActions()) == 0 {
		framework.Fail(ctx, "Policy 'CVSS >= 7' is enabled, but not enforced")
		return
	}

	framework.Pass(ctx, "Policy 'CVSS >= 7' is enabled and enforced")
}
