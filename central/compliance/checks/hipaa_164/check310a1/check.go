package check310a1

import (
	"github.com/stackrox/stackrox/central/compliance/checks/common"
	"github.com/stackrox/stackrox/central/compliance/framework"
	pkgFramework "github.com/stackrox/stackrox/pkg/compliance/framework"
)

const checkID = "HIPAA_164:310_a_1"

func init() {
	framework.MustRegisterNewCheck(
		framework.CheckMetadata{
			ID:                 checkID,
			Scope:              pkgFramework.ClusterKind,
			DataDependencies:   []string{"ImageIntegrations", "NetworkGraph"},
			InterpretationText: interpretationText,
		},
		clusterIsCompliant)
}

func clusterIsCompliant(ctx framework.ComplianceContext) {
	common.CheckImageScannerInUseByCluster(ctx)
	common.CheckNetworkPoliciesByDeployment(ctx)
}
