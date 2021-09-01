package rbac

import (
	"strings"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/k8srbac"
)

type namespacedSubject string

func nsSubjectFromSubject(s *storage.Subject) namespacedSubject {
	b := strings.Builder{}
	name := s.GetName()
	namespace := s.GetNamespace()
	b.Grow(len(namespace) + len(name) + 1)
	b.WriteString(namespace)
	b.WriteString("#")
	b.WriteString(name)
	return namespacedSubject(b.String())
}

type evaluator struct {
	permissionsForSubject map[namespacedSubject]storage.PermissionLevel
}

func (e *evaluator) GetPermissionLevelForSubject(subject *storage.Subject) storage.PermissionLevel {
	level, ok := e.permissionsForSubject[nsSubjectFromSubject(subject)]
	if !ok {
		return storage.PermissionLevel_NONE
	}
	return level
}

func newBucketEvaluator(roles map[namespacedRoleRef]*namespacedRole, bindings map[namespacedBindingID]*namespacedBinding) *evaluator {
	return evaluateRules(groupRulesBySubject(roles, bindings))
}

func groupRulesBySubject(roles map[namespacedRoleRef]*namespacedRole, bindings map[namespacedBindingID]*namespacedBinding) (namespaceSubjectToRules, clusterSubjectToRules map[namespacedSubject]k8srbac.PolicyRuleSet) {
	namespaceSubjectToRules = make(map[namespacedSubject]k8srbac.PolicyRuleSet, len(bindings))
	clusterSubjectToRules = make(map[namespacedSubject]k8srbac.PolicyRuleSet, len(bindings))
	for bID, b := range bindings {
		role, ok := roles[b.roleRef]
		if !ok {
			continue // This roleRef is dangling, no rules for us to use.
		}

		for _, subject := range b.subjects {
			subjectToRules := namespaceSubjectToRules
			if bID.IsClusterBinding() {
				subjectToRules = clusterSubjectToRules
			}
			ruleSet, ok := subjectToRules[subject]
			if !ok {
				ruleSet = k8srbac.NewPolicyRuleSet(k8srbac.CoreFields()...)
				subjectToRules[subject] = ruleSet
			}
			ruleSet.Add(role.rules...)
		}
	}
	return namespaceSubjectToRules, clusterSubjectToRules
}

func evaluateRules(subjectToRules, clusterSubjectToRules map[namespacedSubject]k8srbac.PolicyRuleSet) *evaluator {
	permissionsForSubject := make(map[namespacedSubject]storage.PermissionLevel, len(subjectToRules)+len(clusterSubjectToRules))

	for subject, rules := range clusterSubjectToRules {
		permissionLevel := storage.PermissionLevel_NONE
		if rules.Grants(k8srbac.EffectiveAdmin) {
			permissionLevel = storage.PermissionLevel_CLUSTER_ADMIN
			delete(subjectToRules, subject)
		} else if k8srbac.CanWriteAResource(rules) || k8srbac.CanReadAResource(rules) {
			permissionLevel = storage.PermissionLevel_ELEVATED_CLUSTER_WIDE
			delete(subjectToRules, subject)
		}
		permissionsForSubject[subject] = permissionLevel
	}

	for subject, rules := range subjectToRules {
		permissionLevel := storage.PermissionLevel_NONE
		if k8srbac.CanWriteAResource(rules) || rules.Grants(k8srbac.ListAnything) {
			permissionLevel = storage.PermissionLevel_ELEVATED_IN_NAMESPACE
		} else if k8srbac.CanReadAResource(rules) {
			permissionLevel = storage.PermissionLevel_DEFAULT
		}
		permissionsForSubject[subject] = permissionLevel
	}

	return &evaluator{permissionsForSubject: permissionsForSubject}
}
