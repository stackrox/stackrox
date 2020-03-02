package checkcm5

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/pkg/features"
)

const (
	controlID = "NIST_SP_800_53:CM_5"

	interpretationText = common.IsRBACConfiguredCorrectlyInterpretation + `

` + common.LimitedUsersAndGroupsWithClusterAdminInterpretation
)

func init() {
	framework.MustRegisterNewCheckIfFlagEnabled(
		framework.CheckMetadata{
			ID:                 controlID,
			Scope:              framework.ClusterKind,
			DataDependencies:   []string{"Deployments", "K8sRoles", "K8sRoleBindings"},
			InterpretationText: interpretationText,
		},
		func(ctx framework.ComplianceContext) {
			common.IsRBACConfiguredCorrectly(ctx)
			common.LimitedUsersAndGroupsWithClusterAdmin(ctx)
		}, features.NistSP800_53)
}
