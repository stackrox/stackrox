package checkcm5

import (
	"github.com/stackrox/stackrox/central/compliance/checks/common"
	"github.com/stackrox/stackrox/central/compliance/framework"
	pkgCommon "github.com/stackrox/stackrox/pkg/compliance/checks/common"
	pkgFramework "github.com/stackrox/stackrox/pkg/compliance/framework"
)

const (
	controlID = "NIST_SP_800_53_Rev_4:CM_5"

	interpretationText = pkgCommon.IsRBACConfiguredCorrectlyInterpretation + `

` + common.LimitedUsersAndGroupsWithClusterAdminInterpretation
)

func init() {
	framework.MustRegisterNewCheck(
		framework.CheckMetadata{
			ID:                 controlID,
			Scope:              pkgFramework.ClusterKind,
			DataDependencies:   []string{"Deployments", "K8sRoles", "K8sRoleBindings", "HostScraped"},
			InterpretationText: interpretationText,
		},
		func(ctx framework.ComplianceContext) {
			common.LimitedUsersAndGroupsWithClusterAdmin(ctx)
		})
}
