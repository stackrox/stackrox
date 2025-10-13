package check62

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
	pkgFramework "github.com/stackrox/rox/pkg/compliance/framework"
)

const checkID = "PCI_DSS_3_2:6_2"

func init() {
	framework.MustRegisterNewCheck(
		framework.CheckMetadata{
			ID:                 checkID,
			Scope:              pkgFramework.ClusterKind,
			DataDependencies:   []string{"Images"},
			InterpretationText: interpretationText,
		},
		clusterIsCompliant)
}
func clusterIsCompliant(ctx framework.ComplianceContext) {
	common.CheckFixedCVES(ctx)
	common.CISBenchmarksSatisfied(ctx)
}
