package analysis

import (
	"github.com/stackrox/stackrox/generated/storage"
)

// Analysis holds an analysis of the state of RBAC looking for common mistakes.
type Analysis struct {
	RolesWithoutBindings              []*storage.K8SRole
	BindingsWithoutRoles              []*storage.K8SRoleBinding
	RedundantRoleBindings             map[*storage.K8SRoleBinding]*MatchingRoleBindings
	BindingsForDefaultServiceAccounts []*storage.K8SRoleBinding
	ClusterRolesAggregatingRAndW      []*storage.K8SRoleBinding // TODO(rs): Need to wire in aggregations first.
}

// GetAnalysis returns an Analysis of the state of the RBAC configurations.
func GetAnalysis(roles []*storage.K8SRole, roleBindings []*storage.K8SRoleBinding) *Analysis {
	return &Analysis{
		RolesWithoutBindings:              getRolesWithoutBindings(roles, roleBindings),
		BindingsWithoutRoles:              getBindingsWithoutRoles(roles, roleBindings),
		RedundantRoleBindings:             getRedundantRoleBindings(roleBindings),
		BindingsForDefaultServiceAccounts: getRoleBindingsForDefaultServiceAccounts(roleBindings),
	}
}
