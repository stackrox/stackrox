package check308a5iib

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/generated/storage"
)

const checkID = "HIPAA_164:308_a_5_ii_b"

func init() {
	framework.MustRegisterNewCheck(
		checkID,
		framework.ClusterKind,
		[]string{"Policies"},
		clusterIsCompliant)
}

func clusterIsCompliant(ctx framework.ComplianceContext) {
	common.CheckAnyPolicyInLifeCycle(ctx, storage.LifecycleStage_BUILD)
	common.CheckAnyPolicyInLifeCycle(ctx, storage.LifecycleStage_DEPLOY)
	common.CheckAnyPolicyInLifeCycle(ctx, storage.LifecycleStage_RUNTIME)
}
