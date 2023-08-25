package k8srbac

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
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
		if IsClusterRoleBinding(binding) {
			clusterBindings = append(clusterBindings, binding)
		}
	}
	return NewEvaluator(clusterRoles, clusterBindings)
}
