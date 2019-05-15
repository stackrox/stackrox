package k8srbac

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
)

const clusterAdmin = "cluster-admin"

// Evaluator evaluates the policy rules that apply to different object types.
type Evaluator interface {
	ForSubject(subject *storage.Subject) PolicyRuleSet
	IsClusterAdmin(subject *storage.Subject) bool
	RolesForSubject(subject *storage.Subject) []*storage.K8SRole
}

// NewCombinedEvaluator returns an evaluator that combines the output rules of the two input evaluators.
func NewCombinedEvaluator(e1 Evaluator, e2 Evaluator) Evaluator {
	return &combinedEvaluator{
		e1: e1,
		e2: e2,
	}
}

type combinedEvaluator struct {
	e1 Evaluator
	e2 Evaluator
}

func (e *combinedEvaluator) ForSubject(subject *storage.Subject) PolicyRuleSet {
	ps1 := e.e1.ForSubject(subject)
	ps1.Add(e.e2.ForSubject(subject).ToSlice()...)
	return ps1
}

func (e *combinedEvaluator) IsClusterAdmin(subject *storage.Subject) bool {
	return e.e1.IsClusterAdmin(subject) || e.e2.IsClusterAdmin(subject)
}

func (e *combinedEvaluator) RolesForSubject(subject *storage.Subject) []*storage.K8SRole {
	rolesFromE1 := e.e1.RolesForSubject(subject)
	rolesAdded := set.NewStringSet()
	allRoles := make([]*storage.K8SRole, 0, len(rolesFromE1))
	for _, role := range rolesFromE1 {
		allRoles = append(allRoles, role)
		rolesAdded.Add(role.GetId())
	}

	rolesFromE2 := e.e2.RolesForSubject(subject)
	for _, role := range rolesFromE2 {
		if !rolesAdded.Contains(role.GetId()) {
			allRoles = append(allRoles, role)
		}
	}
	return allRoles
}

// NewEvaluator returns a new instance of an Evaluator.
func NewEvaluator(roles []*storage.K8SRole, bindings []*storage.K8SRoleBinding) Evaluator {
	return &evaluator{
		k8sroles:    roles,
		k8sbindings: bindings,
		bindings:    buildMap(roles, bindings),
	}
}

// MakeClusterEvaluator creates an evaluator for cluster level permissions from the given roles and bindings.
func MakeClusterEvaluator(roles []*storage.K8SRole, bindings []*storage.K8SRoleBinding) Evaluator {
	// Collect cluster roles.
	var clusterRoles []*storage.K8SRole
	for _, role := range roles {
		if role.GetClusterRole() {
			clusterRoles = append(clusterRoles, role)
		}
	}

	// Collect cluster role bindings.
	var clusterBindings []*storage.K8SRoleBinding
	for _, binding := range bindings {
		if binding.GetClusterRole() {
			clusterBindings = append(clusterBindings, binding)
		}
	}
	return NewEvaluator(clusterRoles, clusterBindings)
}

// MakeNamespaceEvaluators creates an evaluator per namespace with a binding present.
func MakeNamespaceEvaluators(roles []*storage.K8SRole, bindings []*storage.K8SRoleBinding) map[string]Evaluator {
	// Collect cluster roles.
	namespaces := set.NewStringSet()
	for _, binding := range bindings {
		namespaces.Add(binding.GetNamespace())
		for _, subject := range binding.GetSubjects() {
			namespaces.Add(subject.GetNamespace())
		}
	}

	evaluators := make(map[string]Evaluator)
	for _, namespace := range namespaces.AsSlice() {
		evaluators[namespace] = MakeNamespaceEvaluator(namespace, roles, bindings)
	}
	return evaluators
}

// MakeNamespaceEvaluator creates an evaluator for the given namespace with the given roles and bindings.
func MakeNamespaceEvaluator(namespace string, roles []*storage.K8SRole, bindings []*storage.K8SRoleBinding) Evaluator {
	// Collect role bindings.
	var namespacedBindings []*storage.K8SRoleBinding
	for _, binding := range bindings {
		if binding.GetClusterRole() || binding.GetNamespace() == namespace {
			namespacedBindings = append(namespacedBindings, binding)
		}
	}
	return NewEvaluator(roles, namespacedBindings)
}
