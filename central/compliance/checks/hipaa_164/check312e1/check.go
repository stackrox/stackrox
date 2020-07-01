package check312e1

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/checks/kubernetes"
	"github.com/stackrox/rox/central/compliance/framework"
)

const (
	standardID = "HIPAA_164:312_e_1"
)

func init() {
	framework.MustRegisterNewCheck(
		framework.CheckMetadata{
			ID:                 standardID,
			Scope:              framework.NodeKind,
			AdditionalScopes:   []framework.TargetKind{framework.DeploymentKind},
			DataDependencies:   []string{"Deployments", "K8sRoles", "K8sRoleBindings", "Policies", "HostScraped"},
			InterpretationText: interpretationText,
		},
		clusterIsCompliant)
}

func clusterIsCompliant(ctx framework.ComplianceContext) {
	kubernetes.MasterAPIServerCommandLine("NIST_SP_800_53_Rev_4", "authorization-mode", "RBAC", "RBAC", common.Contains).Run(ctx)
	common.CheckDeploymentsDoNotHaveClusterAccess(ctx, common.EffectiveAdmin)
	common.CheckNetworkPoliciesByDeployment(ctx)
}
