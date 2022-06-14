package check411

import (
	"github.com/stackrox/stackrox/central/compliance/checks/common"
	"github.com/stackrox/stackrox/central/compliance/framework"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/booleanpolicy/policyfields"
	pkgFramework "github.com/stackrox/stackrox/pkg/compliance/framework"
	pkg "github.com/stackrox/stackrox/pkg/policies"
)

const (
	standardID = "NIST_800_190:4_1_1"
)

func init() {
	framework.MustRegisterNewCheck(
		framework.CheckMetadata{
			ID:                 standardID,
			Scope:              pkgFramework.ClusterKind,
			DataDependencies:   []string{"Policies", "ImageIntegrations"},
			InterpretationText: interpretationText,
		},
		func(ctx framework.ComplianceContext) {
			checkNIST411(ctx)
		})
}

func checkNIST411(ctx framework.ComplianceContext) {
	checkCVSS7PolicyEnforcedOnBuild(ctx)
	checkCriticalVulnPolicyEnforcedOnDeploy(ctx)
	common.CheckImageScannerInUseByCluster(ctx)
	common.CheckAnyPolicyInLifecycleStageEnforced(ctx, storage.LifecycleStage_BUILD)
}

func checkCVSS7PolicyEnforcedOnBuild(ctx framework.ComplianceContext) {
	policiesEnabledNotEnforced := []string{}
	policies := ctx.Data().Policies()
	passed := 0
	for _, p := range policies {
		if (!policyHasCVSS(p) && !policyHasSeverity(p)) || !pkg.AppliesAtBuildTime(p) {
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
		framework.Pass(ctx, "At least one build-stage policy is enabled and enforced that disallows images with a critical vulnerability")
	} else if len(policiesEnabledNotEnforced) > 0 {
		framework.Failf(ctx, "Enforcement is not set on any build-stage policies that disallow images with a critical vulnerability (%v)", policiesEnabledNotEnforced)
	} else {
		framework.Fail(ctx, "No build-stage policy that disallows images with a critical vulnerability was found")
	}
}

func checkCriticalVulnPolicyEnforcedOnDeploy(ctx framework.ComplianceContext) {
	policiesEnabledNotEnforced := []string{}
	policies := ctx.Data().Policies()
	passed := 0
	for _, p := range policies {
		if (!policyHasCVSS(p) && !policyHasSeverity(p)) || !pkg.AppliesAtDeployTime(p) {
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
		framework.Pass(ctx, "Deploy time policies that disallows images with a critical vulnerability is enabled and enforced")
	} else if len(policiesEnabledNotEnforced) > 0 {
		framework.Failf(ctx, "Enforcement is not set on the deploy time policies that disallows images with a critical vulnerability (%v)", policiesEnabledNotEnforced)
	} else {
		framework.Fail(ctx, "No deploy time policy that disallows images with a critical vulnerability was found")
	}
}

func policyHasCVSS(p *storage.Policy) bool {
	return policyfields.ContainsCVSSField(p)
}

func policyHasSeverity(p *storage.Policy) bool {
	return policyfields.ContainsSeverityField(p)
}
