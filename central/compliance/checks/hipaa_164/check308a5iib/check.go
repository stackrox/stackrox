package check308a5iib

import (
	"github.com/stackrox/stackrox/central/compliance/checks/common"
	"github.com/stackrox/stackrox/central/compliance/framework"
	"github.com/stackrox/stackrox/generated/storage"
	pkgFramework "github.com/stackrox/stackrox/pkg/compliance/framework"
)

const checkID = "HIPAA_164:308_a_5_ii_b"

func init() {
	framework.MustRegisterNewCheck(
		framework.CheckMetadata{
			ID:                 checkID,
			Scope:              pkgFramework.ClusterKind,
			DataDependencies:   []string{"Policies"},
			InterpretationText: interpretationText,
		},
		clusterIsCompliant)
}

func clusterIsCompliant(ctx framework.ComplianceContext) {
	common.CheckAnyPolicyInLifeCycle(ctx, storage.LifecycleStage_BUILD)
	common.CheckAnyPolicyInLifeCycle(ctx, storage.LifecycleStage_DEPLOY)
	common.CheckAnyPolicyInLifeCycle(ctx, storage.LifecycleStage_RUNTIME)
}
