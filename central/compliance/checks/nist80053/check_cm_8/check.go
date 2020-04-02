package checkcm8

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
)

const (
	controlID = `NIST_SP_800_53_Rev_4:CM_8`

	interpretationText = `This control requires an up-to-date inventory of information system components.

For this control, ` + common.AllDeployedImagesHaveMatchingIntegrationsInterpretation
)

func init() {
	framework.MustRegisterNewCheck(
		framework.CheckMetadata{
			ID:                 controlID,
			Scope:              framework.ClusterKind,
			DataDependencies:   []string{"Deployments"},
			InterpretationText: interpretationText,
		},
		func(ctx framework.ComplianceContext) {
			common.CheckAllDeployedImagesHaveMatchingIntegrations(ctx)
		})
}
