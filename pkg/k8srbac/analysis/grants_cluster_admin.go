package analysis

import (
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/k8srbac"
	"github.com/stackrox/stackrox/pkg/set"
)

const clusterAdmin = "cluster-admin"

func getSubjectsGrantedClusterAdmin(roles []*storage.K8SRole, roleBindings []*storage.K8SRoleBinding) []*storage.Subject {
	// Collect the id of cluster admin roles. Expected to be 1.
	clusterAdminRoleIDs := set.NewStringSet()
	for _, role := range roles {
		if role.GetName() == clusterAdmin {
			clusterAdminRoleIDs.Add(role.GetId())
		} else if role.GetClusterRole() && grantsAllCoreAPIAccess(role) {
			clusterAdminRoleIDs.Add(role.GetId())
		}
	}
	if clusterAdminRoleIDs.Cardinality() == 0 {
		return nil
	}

	// For every binding that binds to a cluster admin role, collects all of it's subjects.
	subjectsWithClusterAdmin := k8srbac.NewSubjectSet()
	for _, binding := range roleBindings {
		if !k8srbac.IsDefaultRoleBinding(binding) && clusterAdminRoleIDs.Contains(binding.GetRoleId()) {
			subjectsWithClusterAdmin.Add(binding.GetSubjects()...)
		}
	}
	return subjectsWithClusterAdmin.ToSlice()
}

func grantsAllCoreAPIAccess(role *storage.K8SRole) bool {
	ruleSet := k8srbac.NewPolicyRuleSet(k8srbac.CoreFields()...)
	ruleSet.Add(role.GetRules()...)
	return ruleSet.Grants(&storage.PolicyRule{
		ApiGroups: []string{
			"",
		},
		Resources: []string{
			"*",
		},
		Verbs: []string{
			"*",
		},
	})
}
