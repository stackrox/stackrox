package check308a6ii

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/generated/storage"
)

const checkID = "HIPAA_164:308_a_6_ii"

func init() {
	framework.MustRegisterNewCheck(
		framework.CheckMetadata{
			ID:                 checkID,
			Scope:              framework.ClusterKind,
			DataDependencies:   []string{"Notifiers", "Images", "ImageIntegrations", "Policies", "NetworkGraph", "NetworkPolicies"},
			InterpretationText: interpretationText,
		},
		clusterIsCompliant)
}

func clusterIsCompliant(ctx framework.ComplianceContext) {
	common.CheckNotifierInUse(ctx)
	common.IsImageScannerInUse(ctx)
	common.ClusterHasNetworkPolicies(ctx)
	common.CheckFixedCVES(ctx)
	common.CheckAnyPolicyInLifeCycle(ctx, storage.LifecycleStage_RUNTIME)
	common.CheckAnyPolicyInLifeCycle(ctx, storage.LifecycleStage_BUILD)
	common.CheckAnyPolicyInLifeCycle(ctx, storage.LifecycleStage_DEPLOY)
}
