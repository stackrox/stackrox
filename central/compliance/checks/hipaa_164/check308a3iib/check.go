package check308a3iib

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/checks/kubernetes"
	"github.com/stackrox/rox/central/compliance/framework"
)

const (
	standardID = "HIPAA_164:308_a_3_ii_b"
)

func init() {
	framework.MustRegisterNewCheck(
		framework.CheckMetadata{
			ID:                 standardID,
			Scope:              framework.NodeKind,
			AdditionalScopes:   []framework.TargetKind{framework.DeploymentKind},
			DataDependencies:   []string{"Deployments", "K8sRoles", "K8sRoleBindings", "HostScraped"},
			InterpretationText: interpretationText,
		},
		clusterIsCompliant)
}

func clusterIsCompliant(ctx framework.ComplianceContext) {
	kubernetes.MasterAPIServerCommandLine(standardID, "authorization-mode", "RBAC", "RBAC", common.Contains).Run(ctx)
	common.CheckDeploymentsDoNotHaveClusterAccess(ctx, common.EffectiveAdmin)
}
