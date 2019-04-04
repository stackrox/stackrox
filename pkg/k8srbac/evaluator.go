package k8srbac

import (
	"github.com/stackrox/rox/generated/storage"
)

// Evaluator evaluates the policy rules that apply to different object types.
type Evaluator interface {
	ForSubject(subject *storage.Subject) []*storage.PolicyRule
}

// NewEvaluator returns a new instance of an Evaluator.
func NewEvaluator(roles []*storage.K8SRole, bindings []*storage.K8SRoleBinding) Evaluator {
	return &evaluator{
		bindings: buildMap(roles, bindings),
	}
}

type evaluator struct {
	bindings map[SubjectSet]*storage.K8SRole
}

// ForSubject returns the PolicyRules that apply to a subject based on the evaluator's roles and bindings.
func (e *evaluator) ForSubject(subject *storage.Subject) []*storage.PolicyRule {
	// Collect all of the rules for all of the roles that bind the deployment to a role.
	policyRuleSet := NewPolicyRuleSet()
	for subjectSet, role := range e.bindings {
		if subjectSet.Contains(subject) {
			policyRuleSet.Add(role.GetRules()...)
		}
	}
	return policyRuleSet.ToSlice()
}

// Static helper functions.
///////////////////////////

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
