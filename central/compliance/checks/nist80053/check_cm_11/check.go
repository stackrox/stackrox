package checkcm11

import (
	"strings"

	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/pkg/booleanpolicy/policyfields"
	pkgFramework "github.com/stackrox/rox/pkg/compliance/framework"
	"github.com/stackrox/rox/pkg/set"
)

const (
	controlID = `NIST_SP_800_53_Rev_4:CM_11`

	interpretationText = `This control requires monitoring of user-installed software.

For this control, ` + common.AllDeployedImagesHaveMatchingIntegrationsInterpretation + `

StackRox also checks that at least one policy is enabled to check the image registries used in deployments (for example, alerting on images deployed from a public registry), and that built-in package manager execution policies are all enabled.`
)

var (
	defaultRuntimePackageManagementPolicies = []string{
		"d7a275e1-1bba-47e7-92a1-42340c759883", // Ubuntu Package Manager Execution
		"d63564bd-c184-40bc-9f30-39711e010b82", // Alpine Linux Package Manager Execution
		"ddb7af9c-5ec1-45e1-a0cf-c36e3ef2b2ce", // Red Hat Package Manager Execution
	}
)

func checkAtLeastOnePolicyTargetsAnImageRegistry(ctx framework.ComplianceContext) {
	var passed bool
	for _, policy := range ctx.Data().Policies() {
		if !common.IsPolicyEnabled(policy) {
			continue
		}
		var registries []string
		for _, registryField := range policyfields.GetImageRegistries(policy) {
			if registryField != "" {
				registries = append(registries, registryField)
			}
		}
		if len(registries) > 0 {
			passed = true
			var enforcedText string
			if len(policy.GetEnforcementActions()) > 0 {
				enforcedText = " (which is enforced)"
			}

			framework.Passf(ctx, "Policy %q%s targets image registries (%s)", policy.GetName(), enforcedText, strings.Join(registries, ", "))
		}
	}
	if !passed {
		framework.Fail(ctx, "No active policy targets image registries")
	}
}

func checkAllDefaultRuntimePackageManagementPoliciesEnabled(ctx framework.ComplianceContext) {
	unseenPolicies := set.NewStringSet(defaultRuntimePackageManagementPolicies...)
	for _, policy := range ctx.Data().Policies() {
		if !common.IsPolicyEnabled(policy) {
			continue
		}
		unseenPolicies.Remove(policy.GetId())
	}
	if unseenPolicies.Cardinality() > 0 {
		framework.Failf(ctx, "Built-in runtime package management policies %s not enabled", strings.Join(unseenPolicies.AsSlice(), ", "))
	} else {
		framework.Pass(ctx, "All built-in runtime package management policies are enabled")
	}
}

func init() {
	framework.MustRegisterNewCheck(
		framework.CheckMetadata{
			ID:                 controlID,
			Scope:              pkgFramework.ClusterKind,
			DataDependencies:   []string{"Policies", "Deployments"},
			InterpretationText: interpretationText,
		},
		func(ctx framework.ComplianceContext) {
			checkAtLeastOnePolicyTargetsAnImageRegistry(ctx)
			common.CheckAllDeployedImagesHaveMatchingIntegrations(ctx)
			checkAllDefaultRuntimePackageManagementPoliciesEnabled(ctx)
		})
}
