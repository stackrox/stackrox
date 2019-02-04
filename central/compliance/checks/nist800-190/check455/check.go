package check455

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/generated/storage"
)

const (
	standardID = "NIST_800_190:4_5_5"
)

func init() {
	framework.MustRegisterNewCheck(
		framework.CheckMetadata{
			ID:                 standardID,
			Scope:              framework.DeploymentKind,
			DataDependencies:   []string{"Deployments"},
			InterpretationText: interpretationText,
		},
		checkNIST455)
}

func checkNIST455(ctx framework.ComplianceContext) {
	checkDeploymentHostMounts(ctx)
}

func checkDeploymentHostMounts(ctx framework.ComplianceContext) {
	framework.ForEachDeployment(ctx, func(ctx framework.ComplianceContext, deployment *storage.Deployment) {
		if common.DeploymentHasHostMounts(deployment) {
			framework.Failf(ctx, "Deployment %s is using host mounts.", deployment.GetName())
		} else {
			framework.Passf(ctx, "Deployment %s has no host mounts.", deployment.GetName())
		}
	})
}
