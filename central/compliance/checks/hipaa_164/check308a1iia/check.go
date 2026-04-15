package check308a1iia

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
	pkgFramework "github.com/stackrox/rox/pkg/compliance/framework"
)

const checkID = "HIPAA_164:308_a_1_ii_a"

func Register308a1iia() {
	framework.MustRegisterNewCheck(
		framework.CheckMetadata{
			ID:                 checkID,
			Scope:              pkgFramework.ClusterKind,
			DataDependencies:   []string{"ImageIntegrations"},
			InterpretationText: interpretationText,
		},
		clusterIsCompliant)
}

func clusterIsCompliant(ctx framework.ComplianceContext) {
	common.CheckImageScannerInUseByCluster(ctx)
	common.CheckImageScannerWasUsed(ctx)
	common.CheckRuntimeSupportInCluster(ctx)
}
