package checkra3

import (
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/pkg/features"
)

const (
	controlID = "NIST_SP_800_53:RA_3"

	interpretationText = `TODO`
)

func init() {
	framework.MustRegisterNewCheckIfFlagEnabled(
		framework.CheckMetadata{
			ID:                 controlID,
			Scope:              framework.ClusterKind,
			DataDependencies:   []string{"Cluster"},
			InterpretationText: interpretationText,
		},
		func(ctx framework.ComplianceContext) {
			framework.Passf(ctx, "StackRox is installed in cluster %q, and provides continuous risk assessment.", ctx.Data().Cluster().GetName())
		}, features.NistSP800_53)
}
