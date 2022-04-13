package checkcm6

import (
	"github.com/stackrox/stackrox/central/compliance/checks/common"
	"github.com/stackrox/stackrox/central/compliance/framework"
	pkgFramework "github.com/stackrox/stackrox/pkg/compliance/framework"
)

const (
	controlID = "NIST_SP_800_53_Rev_4:CM_6"

	interpretationText = `This control requires that configuration controls be implemented and deviations are documented.

For this control, ` + common.CheckNoViolationsForDeployPhasePoliciesInterpretation + `

To approve a deviation, resolve the policy violation or adjust the scope or exclusions for the policy.`
)

func init() {
	framework.MustRegisterNewCheck(
		framework.CheckMetadata{
			ID:                 controlID,
			Scope:              pkgFramework.ClusterKind,
			DataDependencies:   []string{"Policies", "UnresolvedAlerts"},
			InterpretationText: interpretationText,
		},
		func(ctx framework.ComplianceContext) {
			common.CheckNoViolationsForDeployPhasePolicies(ctx)
		})
}
