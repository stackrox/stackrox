package checkra3

import (
	"github.com/stackrox/rox/central/compliance/framework"
	pkgFramework "github.com/stackrox/rox/pkg/compliance/framework"
)

const (
	controlID = "NIST_SP_800_53_Rev_4:RA_3"

	interpretationText = `This control requires ongoing risk assessment.

For this control, StackRox checks that StackRox components are installed in each cluster, providing continuous multi-factor risk assessment.`
)

func init() {
	framework.MustRegisterNewCheck(
		framework.CheckMetadata{
			ID:                 controlID,
			Scope:              pkgFramework.ClusterKind,
			DataDependencies:   []string{"Cluster"},
			InterpretationText: interpretationText,
		},
		func(ctx framework.ComplianceContext) {
			framework.Passf(ctx, "StackRox is installed in cluster %q, and provides continuous risk assessment.", ctx.Data().Cluster().GetName())
		})
}
