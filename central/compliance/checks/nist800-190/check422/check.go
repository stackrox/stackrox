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
		framework.CheckMetadata{
			ID:                 standardID,
			Scope:              framework.ClusterKind,
			DataDependencies:   []string{"Policies", "ImageIntegrations"},
			InterpretationText: interpretationText,
		},
		func(ctx framework.ComplianceContext) {
			checkNIST422(ctx)
		})
}

func checkNIST422(ctx framework.ComplianceContext) {
	checkLatestImageTagPolicyEnforced(ctx)
	checkImageAgePolicyEnforced(ctx)
}

func checkImageAgePolicyEnforced(ctx framework.ComplianceContext) {
	policiesEnabledNotEnforced := []string{}
	policies := ctx.Data().Policies()
	passed := 0
	for _, p := range policies {
		if !policyHasImageAgeDays(p) {
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
		framework.Passf(ctx, "Policy that disallows old images to be deployed is enabled and enforced")
	} else if len(policiesEnabledNotEnforced) > 0 {
		framework.Failf(ctx, "Enforcement is not set on the policies that disallow old images to be deployed (%v)", policiesEnabledNotEnforced)
	} else {
		framework.Fail(ctx, "Policy that disallows old images to be deployed not found")
	}
}

func checkLatestImageTagPolicyEnforced(ctx framework.ComplianceContext) {
	policiesEnabledNotEnforced := []string{}
	policies := ctx.Data().Policies()
	passed := 0
	for _, p := range policies {
		if !policyHasLatestImageTag(p) {
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
		framework.Passf(ctx, "Policy that disallows images with tag 'latest' to be deployed is enabled and enforced")
	} else if len(policiesEnabledNotEnforced) > 0 {
		framework.Failf(ctx, "Enforcement is not set on the policies that disallow images with tag 'latest' to be deployed (%v)", policiesEnabledNotEnforced)
	} else {
		framework.Fail(ctx, "Policy that disallows images with tag 'latest' to be deployed was not found")
	}
}

func policyHasLatestImageTag(p *storage.Policy) bool {
	return p.GetFields().GetImageName().GetTag() == "latest"
}

func policyHasImageAgeDays(p *storage.Policy) bool {
	return p.GetFields().GetSetImageAgeDays() != nil
}
