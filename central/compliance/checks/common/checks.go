package common

import (
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/generated/storage"
)

// CheckImageScannerInUse checks if we have atleast one image scanner in place.
func CheckImageScannerInUse(ctx framework.ComplianceContext) {
	var scanners []string
	for _, integration := range ctx.Data().ImageIntegrations() {
		for _, category := range integration.GetCategories() {
			if category == storage.ImageIntegrationCategory_SCANNER {
				scanners = append(scanners, integration.Name)
			}
		}
	}

	if len(scanners) > 0 {
		if len(scanners) == 1 {
			framework.Passf(ctx, "An image vulnerability scanner (%s) is configured", scanners[0])
		} else {
			framework.Passf(ctx, "%d image vulnerability scanners are configured", len(scanners))
		}
	} else {
		framework.Failf(ctx, "No image vulnerability scanners are configured")
	}
}

// CheckBuildTimePolicyEnforced checks if any build time policies are being enforced.
func CheckBuildTimePolicyEnforced(ctx framework.ComplianceContext) {
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

// CheckPolicyInUse checks if a policy is in use.
func CheckPolicyInUse(ctx framework.ComplianceContext, name string) {
	policies := ctx.Data().Policies()
	p := policies[name]

	if p.GetDisabled() {
		framework.Failf(ctx, "'%s' policy is not in use", name)
		return
	}

	framework.Passf(ctx, "'%s' policy is in use", name)
}

// CheckPolicyEnforced checks if a policy is in use and is enforced.
func CheckPolicyEnforced(ctx framework.ComplianceContext, name string) {
	policies := ctx.Data().Policies()
	p := policies[name]

	if p.GetDisabled() {
		framework.Failf(ctx, "'%s' policy is not in use", name)
		return
	}

	if len(p.GetEnforcementActions()) == 0 {
		framework.Failf(ctx, "'%s' policy is not being enforced", name)
		return
	}

	framework.Passf(ctx, "'%s' policy is in use and is enforced", name)
}

// CheckAnyPolicyInCategory checks if there are any enabled policies in the given category.
func CheckAnyPolicyInCategory(ctx framework.ComplianceContext, category string) {
	categoryPolicies := ctx.Data().PolicyCategories()
	policySet := categoryPolicies[category]
	if policySet.Cardinality() == 0 {
		framework.Failf(ctx, "No policies are in place to monitor '%s' category issues", category)
		return
	}
	framework.Passf(ctx, "Policies are in place to monitor '%s' category issues", category)
}

// CheckAnyPolicyInCategoryEnforced checks if there are any enabled policies in the given category and are enforced.
func CheckAnyPolicyInCategoryEnforced(ctx framework.ComplianceContext, category string) {
	categoryPolicies := ctx.Data().PolicyCategories()
	policySet := categoryPolicies[category]
	if policySet.Cardinality() == 0 {
		framework.Failf(ctx, "No policies are in place to monitor '%s' category issues", category)
		return
	}
	policies := policySet.AsSlice()
	for _, policy := range policies {
		CheckPolicyEnforced(ctx, policy)
	}
}

// DeploymentHasHostMounts returns true if the deployment has host mounts.
func DeploymentHasHostMounts(deployment *storage.Deployment) bool {
	for _, container := range deployment.Containers {
		for _, vol := range container.Volumes {
			if vol.Type == "HostPath" {
				return true
			}
		}
	}
	return false
}
