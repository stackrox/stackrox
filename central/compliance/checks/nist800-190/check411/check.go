package check411

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/generated/storage"
	pkg "github.com/stackrox/rox/pkg/policies"
)

const (
	standardID = "NIST_800_190:4_1_1"
)

func init() {
	framework.MustRegisterNewCheck(
		framework.CheckMetadata{
			ID:                 standardID,
			Scope:              framework.ClusterKind,
			DataDependencies:   []string{"Policies", "ImageIntegrations"},
			InterpretationText: interpretationText,
		},
		func(ctx framework.ComplianceContext) {
			checkNIST411(ctx)
		})
}

func checkNIST411(ctx framework.ComplianceContext) {
	checkCVSS7PolicyEnforcedOnBuild(ctx)
	checkCVSS7PolicyEnforcedOnDeploy(ctx)
	common.CheckImageScannerInUseByCluster(ctx)
	common.CheckBuildTimePolicyEnforced(ctx)
}

func checkCVSS7PolicyEnforcedOnBuild(ctx framework.ComplianceContext) {
	policiesEnabledNotEnforced := []string{}
	policies := ctx.Data().Policies()
	passed := 0
	for _, p := range policies {
		if !policyHasCVSS(p) || !pkg.AppliesAtBuildTime(p) {
			continue
		}
		enabled := common.IsPolicyEnabled(p)
		enforced := common.IsPolicyEnforced(p)
		if enabled && !enforced {
			policiesEnabledNotEnforced = append(policiesEnabledNotEnforced, p.GetName())
			continue
		}

		if enabled && enforced {
			passed++
		}
	}
	if passed >= 1 {
		framework.Pass(ctx, "Build time policies that disallows images with a critical CVSS score is enabled and enforced")
	} else if len(policiesEnabledNotEnforced) > 0 {
		framework.Failf(ctx, "Enforcement is not set on the build time policies that disallows images with a critical CVSS score (%v)", policiesEnabledNotEnforced)
	} else {
		framework.Fail(ctx, "No build time policy that disallows images with a critical CVSS score was found")
	}
}

func checkCVSS7PolicyEnforcedOnDeploy(ctx framework.ComplianceContext) {
	policiesEnabledNotEnforced := []string{}
	policies := ctx.Data().Policies()
	passed := 0
	for _, p := range policies {
		if !policyHasCVSS(p) || !pkg.AppliesAtDeployTime(p) {
			continue
		}
		enabled := common.IsPolicyEnabled(p)
		enforced := common.IsPolicyEnforced(p)
		if enabled && !enforced {
			policiesEnabledNotEnforced = append(policiesEnabledNotEnforced, p.GetName())
			continue
		}

		if enabled && enforced {
			passed++
		}
	}
	if passed >= 1 {
		framework.Pass(ctx, "Deploy time policies that disallows images with a critical CVSS score is enabled and enforced")
	} else if len(policiesEnabledNotEnforced) > 0 {
		framework.Failf(ctx, "Enforcement is not set on the deploy time policies that disallows images with a critical CVSS score (%v)", policiesEnabledNotEnforced)
	} else {
		framework.Fail(ctx, "No deploy time policy that disallows images with a critical CVSS score was found")
	}
}

func policyHasCVSS(p *storage.Policy) bool {
	return p.GetFields().GetCvss() != nil
}
