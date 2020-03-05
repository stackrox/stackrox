package checkac24

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/pkg/features"
)

const (
	controlID = "NIST_SP_800_53_Rev_4:AC_24"

	interpretationText = common.IsRBACConfiguredCorrectlyInterpretation
)

func init() {
	framework.MustRegisterNewCheckIfFlagEnabled(
		framework.CheckMetadata{
			ID:                 controlID,
			Scope:              framework.ClusterKind,
			DataDependencies:   []string{"Deployments"},
			InterpretationText: interpretationText,
		},
		func(ctx framework.ComplianceContext) {
			common.IsRBACConfiguredCorrectly(ctx)
		}, features.NistSP800_53)
}
