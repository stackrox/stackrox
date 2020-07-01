package check85

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/checks/kubernetes"
	"github.com/stackrox/rox/central/compliance/framework"
)

const (
	standardID = "PCI_DSS_3_2"
	checkID    = standardID + ":8_5"
)

func init() {
	framework.MustRegisterNewCheck(
		framework.CheckMetadata{
			ID:                 checkID,
			Scope:              framework.ClusterKind,
			AdditionalScopes:   []framework.TargetKind{framework.DeploymentKind},
			DataDependencies:   []string{"Deployments", "K8sRoles", "K8sRoleBindings"},
			InterpretationText: interpretationText,
		},
		clusterIsCompliant)
}

func clusterIsCompliant(ctx framework.ComplianceContext) {
	kubernetes.MasterAPIServerCommandLine(standardID, "authorization-mode", "RBAC", "RBAC", common.Contains).Run(ctx)
	common.LimitedUsersAndGroupsWithClusterAdmin(ctx)
	common.AdministratorUsersPresent(ctx)
	common.CheckDeploymentsDoNotHaveClusterAccess(ctx, common.EffectiveAdmin)
}
