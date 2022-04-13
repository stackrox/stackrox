package check308a1iib

import (
	"github.com/stackrox/stackrox/central/compliance/checks/common"
	"github.com/stackrox/stackrox/central/compliance/framework"
	pkgFramework "github.com/stackrox/stackrox/pkg/compliance/framework"
)

const checkID = "HIPAA_164:308_a_1_ii_b"

func init() {
	framework.MustRegisterNewCheck(
		framework.CheckMetadata{
			ID:                 checkID,
			Scope:              pkgFramework.ClusterKind,
			DataDependencies:   []string{"ImageIntegrations"},
			InterpretationText: interpretationText,
		},
		common.CheckImageScannerInUseByCluster)
}
