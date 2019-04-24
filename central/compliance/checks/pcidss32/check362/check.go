package check362

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
)

const checkID = "PCI_DSS_3_2:3_6_2"

func init() {
	if features.K8sRBAC.Enabled() {
		framework.MustRegisterNewCheck(
			framework.CheckMetadata{
				ID:                 checkID,
				Scope:              framework.ClusterKind,
				AdditionalScopes:   []framework.TargetKind{framework.DeploymentKind},
				DataDependencies:   []string{"Deployments", "K8sRoles", "K8sRoleBindings", "Policies"},
				InterpretationText: interpretationText,
			},
			clusterIsCompliant)
	} else {
		framework.MustRegisterNewCheck(
			framework.CheckMetadata{
				ID:                 checkID,
				Scope:              framework.ClusterKind,
				DataDependencies:   []string{"Policies"},
				InterpretationText: interpretationText,
			},
			clusterIsCompliant)
	}
}

func clusterIsCompliant(ctx framework.ComplianceContext) {
	if features.K8sRBAC.Enabled() {
		common.IsRBACConfiguredCorrectly(ctx)
		common.CheckDeploymentsDoNotHaveClusterAccess(ctx, common.EffectiveAdmin)
		common.CheckDeploymentsDoNotHaveClusterAccess(ctx, &storage.PolicyRule{
			Verbs:     []string{"*"},
			ApiGroups: []string{""},
			Resources: []string{"secrets"},
		})
	}
	common.CheckSecretsInEnv(ctx)
}
