package k8srbac

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
)

type evaluator struct {
	k8sroles    []*storage.K8SRole
	k8sbindings []*storage.K8SRoleBinding
	bindings    map[SubjectSet]*storage.K8SRole
}

// ForSubject returns the PolicyRules that apply to a subject based on the evaluator's roles and bindings.
func (e *evaluator) ForSubject(subject *storage.Subject) PolicyRuleSet {
	// Collect all of the rules for all of the roles that bind the deployment to a role.
	policyRuleSet := NewPolicyRuleSet(CoreFields()...)
	for subjectSet, role := range e.bindings {
		if subjectSet.Contains(subject) {
			policyRuleSet.Add(role.Clone().GetRules()...)
		}
	}
	return policyRuleSet
}

// IsClusterAdmin returns true if the subject has cluster admin privs, false otherwise
func (e *evaluator) IsClusterAdmin(subject *storage.Subject) bool {
	return isSubjectClusterAdmin(subject, e.k8sroles, e.k8sbindings)
}

// RolesForSubject returns the roles assigned to the subject based on the evaluator's bindings
func (e *evaluator) RolesForSubject(subject *storage.Subject) []*storage.K8SRole {
	// Collect all of the rules for all of the roles that bind the deployment to a role.
	roles := make([]*storage.K8SRole, 0)
	for subjectSet, role := range e.bindings {
		if subjectSet.Contains(subject) {
			roles = append(roles, role.Clone())
		}
	}
	return roles
}

// Static helper functions.
///////////////////////////

func subjectsAreEqual(subject1 *storage.Subject, subject2 *storage.Subject) bool {
	return subject1.GetKind() == subject2.GetKind() &&
		subject1.GetName() == subject2.GetName() &&
		subject1.GetNamespace() == subject2.GetNamespace()
}

func getSubjectsGrantedClusterAdmin(roles []*storage.K8SRole, roleBindings []*storage.K8SRoleBinding) []*storage.Subject {
	// Collect the id of cluster admin roles. Expected to be 1.
	clusterAdminRoleIDs := getClusterAdminRoles(roles)
	if clusterAdminRoleIDs.Cardinality() == 0 {
		return nil
	}
	// For every binding that binds to a cluster admin role, collects all of it's subjects.
	subjectsWithClusterAdmin := NewSubjectSet()
	for _, binding := range roleBindings {
		if !IsDefaultRoleBinding(binding) && clusterAdminRoleIDs.Contains(binding.GetRoleId()) {
			subjectsWithClusterAdmin.Add(binding.GetSubjects()...)
		}
	}
	return subjectsWithClusterAdmin.ToSlice()
}

func isSubjectClusterAdmin(subject *storage.Subject, roles []*storage.K8SRole, roleBindings []*storage.K8SRoleBinding) bool {
	clusterAdminRoleIDs := getClusterAdminRoles(roles)
	subjectsWithClusterAdmin := NewSubjectSet()
	for _, binding := range roleBindings {
		if clusterAdminRoleIDs.Contains(binding.GetRoleId()) {
			subjectsWithClusterAdmin.Add(binding.GetSubjects()...)
		}
	}
	for _, s := range subjectsWithClusterAdmin.ToSlice() {
		if subjectsAreEqual(subject, s) {
			return true
		}
	}
	return false
}

func getClusterAdminRoles(roles []*storage.K8SRole) set.StringSet {
	clusterAdminRoleIDs := set.NewStringSet()
	for _, role := range roles {
		if role.GetName() == clusterAdmin {
			clusterAdminRoleIDs.Add(role.GetId())
		} else if role.GetClusterRole() && grantsAllCoreAPIAccess(role) {
			clusterAdminRoleIDs.Add(role.GetId())
		}
	}
	return clusterAdminRoleIDs
}

func grantsAllCoreAPIAccess(role *storage.K8SRole) bool {
	ruleSet := NewPolicyRuleSet(CoreFields()...)
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

func buildMap(roles []*storage.K8SRole, bindings []*storage.K8SRoleBinding) map[SubjectSet]*storage.K8SRole {
	// Map role id to all of the subjects granted that role.
	roleIDToSubjects := make(map[string]SubjectSet)
	for _, binding := range bindings {
		if _, hasEntry := roleIDToSubjects[binding.GetRoleId()]; !hasEntry {
			roleIDToSubjects[binding.GetRoleId()] = NewSubjectSet()
		}
		roleIDToSubjects[binding.GetRoleId()].Add(binding.GetSubjects()...)
	}

	// Complete the map so that we can test subjects and get the role for it.
	subjectsToRole := make(map[SubjectSet]*storage.K8SRole)
	for _, role := range roles {
		if subjectSet, hasEntry := roleIDToSubjects[role.GetId()]; hasEntry {
			subjectsWithRole := subjectSet
			subjectsToRole[subjectsWithRole] = role
		}
	}
	return subjectsToRole
}
