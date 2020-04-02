package checkac24

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
)

const (
	controlID = "NIST_SP_800_53_Rev_4:AC_24"

	interpretationText = common.IsRBACConfiguredCorrectlyInterpretation
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
			common.IsRBACConfiguredCorrectly(ctx)
		})
}
