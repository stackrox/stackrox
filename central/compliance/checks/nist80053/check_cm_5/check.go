package checkcm5

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
)

const (
	controlID = "NIST_SP_800_53_Rev_4:CM_5"

	interpretationText = common.IsRBACConfiguredCorrectlyInterpretation + `

` + common.LimitedUsersAndGroupsWithClusterAdminInterpretation
)

func init() {
	framework.MustRegisterNewCheck(
		framework.CheckMetadata{
			ID:                 controlID,
			Scope:              framework.ClusterKind,
			DataDependencies:   []string{"Deployments", "K8sRoles", "K8sRoleBindings"},
			InterpretationText: interpretationText,
		},
		func(ctx framework.ComplianceContext) {
			common.IsRBACConfiguredCorrectly(ctx)
			common.LimitedUsersAndGroupsWithClusterAdmin(ctx)
		})
}
