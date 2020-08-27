package check713

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
	pkgFramework "github.com/stackrox/rox/pkg/compliance/framework"
)

const checkID = "PCI_DSS_3_2:7_1_3"

func init() {
	framework.MustRegisterNewCheck(
		framework.CheckMetadata{
			ID:                 checkID,
			Scope:              pkgFramework.ClusterKind,
			AdditionalScopes:   []pkgFramework.TargetKind{pkgFramework.DeploymentKind},
			DataDependencies:   []string{"Deployments", "K8sRoles", "K8sRoleBindings", "HostScraped"},
			InterpretationText: interpretationText,
		},
		clusterIsCompliant)
}

func clusterIsCompliant(ctx framework.ComplianceContext) {
	common.AdministratorUsersPresent(ctx)
	common.CheckDeploymentsDoNotHaveClusterAccess(ctx, common.EffectiveAdmin)
}
