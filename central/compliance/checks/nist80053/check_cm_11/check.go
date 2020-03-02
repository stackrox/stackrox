package checkcm11

import (
	"strings"

	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/set"
)

const (
	controlID = `NIST_SP_800_53:CM_11`

	interpretationText = `This control requires monitoring of user-installed software.

For this control, ` + common.AllDeployedImagesHaveMatchingIntegrationsInterpretation + `

StackRox also checks that at least one policy is enabled for image registries (for example, alerting on images from a public registry), and that built-in package manager execution policies are all enabled.`
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
		if registryField := policy.GetFields().GetImageName().GetRegistry(); registryField != "" {
			passed = true
			var enforcedText string
			if len(policy.GetEnforcementActions()) > 0 {
				enforcedText = " (which is enforced)"
			}

			framework.Passf(ctx, "Policy %q%s targets an image registry (%s)", policy.GetName(), enforcedText, registryField)
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
	framework.MustRegisterNewCheckIfFlagEnabled(
		framework.CheckMetadata{
			ID:                 controlID,
			Scope:              framework.ClusterKind,
			DataDependencies:   []string{"Policies", "Deployments"},
			InterpretationText: interpretationText,
		},
		func(ctx framework.ComplianceContext) {
			checkAtLeastOnePolicyTargetsAnImageRegistry(ctx)
			common.CheckAllDeployedImagesHaveMatchingIntegrations(ctx)
			checkAllDefaultRuntimePackageManagementPoliciesEnabled(ctx)
		}, features.NistSP800_53)
}
