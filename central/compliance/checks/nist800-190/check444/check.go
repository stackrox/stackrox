package check444

import (
	"github.com/stackrox/stackrox/central/compliance/checks/common"
	"github.com/stackrox/stackrox/central/compliance/framework"
	"github.com/stackrox/stackrox/generated/storage"
	pkgFramework "github.com/stackrox/stackrox/pkg/compliance/framework"
)

const (
	standardID = "NIST_800_190:4_4_4"
)

func init() {
	framework.MustRegisterNewCheck(
		framework.CheckMetadata{
			ID:                 standardID,
			Scope:              pkgFramework.ClusterKind,
			AdditionalScopes:   []pkgFramework.TargetKind{pkgFramework.DeploymentKind},
			DataDependencies:   []string{"Policies", "Deployments"},
			InterpretationText: interpretationText,
		},
		checkNIST444)
}

func checkNIST444(ctx framework.ComplianceContext) {
	common.CheckAnyPolicyInLifeCycle(ctx, storage.LifecycleStage_RUNTIME)
	common.CheckDeploymentHasReadOnlyFSByDeployment(ctx)
}
