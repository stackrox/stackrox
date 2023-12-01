package check23

import (
	"github.com/stackrox/rox/central/compliance/framework"
	pkgFramework "github.com/stackrox/rox/pkg/compliance/framework"
)

const checkID = "PCI_DSS_3_2:2_3"

func init() {
	framework.MustRegisterNewCheck(
		framework.CheckMetadata{
			ID:                 checkID,
			Scope:              pkgFramework.ClusterKind,
			InterpretationText: interpretationText,
		},
		clusterIsCompliant)
}

func clusterIsCompliant(ctx framework.ComplianceContext) {
	framework.Pass(ctx, passText())
}
