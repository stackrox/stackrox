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

func newBucketEvaluator(roles map[namespacedRoleRef]*storage.K8SRole, bindings map[string]*storage.K8SRoleBinding) *evaluator {
	return evaluateRules(groupRulesBySubject(bindings, extractRulesByRoleID(roles)))
}

func extractRulesByRoleID(roles map[namespacedRoleRef]*storage.K8SRole) map[string][]*storage.PolicyRule {
	rulesForRoleID := make(map[string][]*storage.PolicyRule, len(roles))
	for _, r := range roles {
		rulesForRoleID[r.GetId()] = r.GetRules()
	}
	return rulesForRoleID
}

func groupRulesBySubject(bindings map[string]*storage.K8SRoleBinding, rulesForRoleID map[string][]*storage.PolicyRule) (namespaceSubjectToRules, clusterSubjectToRules map[namespacedSubject]k8srbac.PolicyRuleSet) {
	namespaceSubjectToRules = make(map[namespacedSubject]k8srbac.PolicyRuleSet, len(bindings))
	clusterSubjectToRules = make(map[namespacedSubject]k8srbac.PolicyRuleSet, len(bindings))
	for _, b := range bindings {
		rules := rulesForRoleID[b.GetRoleId()]
		for _, s := range b.GetSubjects() {
			subject := nsSubjectFromSubject(s)
			subjectToRules := namespaceSubjectToRules
			if b.GetClusterRole() {
				subjectToRules = clusterSubjectToRules
			}
			ruleSet, ok := subjectToRules[subject]
			if !ok {
				ruleSet = k8srbac.NewPolicyRuleSet(k8srbac.CoreFields()...)
				subjectToRules[subject] = ruleSet
			}
			for _, r := range rules {
				ruleSet.Add(r)
			}
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
