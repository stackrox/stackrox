package checksi22

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
	pkgFramework "github.com/stackrox/rox/pkg/compliance/framework"
)

const (
	controlID = `NIST_SP_800_53_Rev_4:SI_2_(2)`

	interpretationText = `This control requires that system flaws be identified and remediated in a timely manner.

For this control, ` + common.AllDeployedImagesHaveMatchingIntegrationsInterpretation + `

Also, ` + common.CheckAtLeastOnePolicyEnabledReferringToVulnsInterpretation
)

func init() {
	framework.MustRegisterNewCheck(
		framework.CheckMetadata{
			ID:                 controlID,
			Scope:              pkgFramework.ClusterKind,
			DataDependencies:   []string{"Policies"},
			InterpretationText: interpretationText,
		},
		func(ctx framework.ComplianceContext) {
			common.CheckAllDeployedImagesHaveMatchingIntegrations(ctx)
			common.CheckAtLeastOnePolicyEnabledReferringToVulns(ctx)
		})
}
