package k8srbac

import (
	"context"

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

// EvaluatorForContext evaluates the policy rules that apply to different object types and takes in a context for permissions.
type EvaluatorForContext interface {
	ForSubject(ctx context.Context, subject *storage.Subject) PolicyRuleSet
	IsClusterAdmin(ctx context.Context, subject *storage.Subject) bool
	RolesForSubject(ctx context.Context, subject *storage.Subject) []*storage.K8SRole
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
	for namespace := range namespaces {
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
