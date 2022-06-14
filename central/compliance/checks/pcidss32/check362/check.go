package check362

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/generated/storage"
	pkgFramework "github.com/stackrox/rox/pkg/compliance/framework"
)

const checkID = "PCI_DSS_3_2:3_6_2"

func init() {
	framework.MustRegisterNewCheck(
		framework.CheckMetadata{
			ID:                 checkID,
			Scope:              pkgFramework.ClusterKind,
			AdditionalScopes:   []pkgFramework.TargetKind{pkgFramework.DeploymentKind},
			DataDependencies:   []string{"Deployments", "K8sRoles", "K8sRoleBindings", "Policies", "HostScraped"},
			InterpretationText: interpretationText,
		},
		clusterIsCompliant)
}

func clusterIsCompliant(ctx framework.ComplianceContext) {
	common.CheckDeploymentsDoNotHaveClusterAccess(ctx, common.EffectiveAdmin)
	common.CheckDeploymentsDoNotHaveClusterAccess(ctx, &storage.PolicyRule{
		Verbs:     []string{"*"},
		ApiGroups: []string{""},
		Resources: []string{"secrets"},
	})
	common.CheckSecretsInEnv(ctx)
}
