package checkra5

import (
	"github.com/stackrox/stackrox/central/compliance/checks/common"
	"github.com/stackrox/stackrox/central/compliance/framework"
	pkgFramework "github.com/stackrox/stackrox/pkg/compliance/framework"
	"github.com/stackrox/stackrox/pkg/set"
)

const (
	controlID = "NIST_SP_800_53_Rev_4:RA_5"

	interpretationText = `This control requires vulnerability scanning and associated workflows.

For this control, ` + common.AllDeployedImagesHaveMatchingIntegrationsInterpretation + `

Also, ` + common.CheckAtLeastOnePolicyEnabledReferringToVulnsInterpretation + `

Further, StackRox verifies that there are no active violations for any of these policies.`
)

func checkNoUnresolvedAlertsForPolicies(ctx framework.ComplianceContext, policyIDs set.StringSet) {
	var foundUnresolvedAlerts bool
	for _, alert := range ctx.Data().UnresolvedAlerts() {
		if policyIDs.Contains(alert.GetPolicy().GetId()) {
			framework.Failf(ctx, "Policy %s refers to vulnerabilities, but has an active violation in deployment %s/%s.",
				alert.GetPolicy().GetName(), alert.GetDeployment().GetNamespace(), alert.GetDeployment().GetName())
			foundUnresolvedAlerts = true
		}
	}
	if !foundUnresolvedAlerts {
		framework.Pass(ctx, "There are no unresolved violations for vulnerability-related policies.")
	}
}

func init() {
	framework.MustRegisterNewCheck(
		framework.CheckMetadata{
			ID:                 controlID,
			Scope:              pkgFramework.ClusterKind,
			DataDependencies:   []string{"UnresolvedAlerts", "Deployments", "Policies"},
			InterpretationText: interpretationText,
		},
		func(ctx framework.ComplianceContext) {
			common.CheckAllDeployedImagesHaveMatchingIntegrations(ctx)
			vulnPolicyIDs := common.CheckAtLeastOnePolicyEnabledReferringToVulns(ctx)
			checkNoUnresolvedAlertsForPolicies(ctx, vulnPolicyIDs)
		})
}
