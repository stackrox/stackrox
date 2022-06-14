package utils

import (
	"context"

	roleStore "github.com/stackrox/stackrox/central/rbac/k8srole/datastore"
	bindingStore "github.com/stackrox/stackrox/central/rbac/k8srolebinding/datastore"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/k8srbac"
	"github.com/stackrox/stackrox/pkg/search"
)

type namespacePermissionEvaluator struct {
	clusterID     string
	namespace     string
	roleStore     roleStore.DataStore
	bindingsStore bindingStore.DataStore
}

// NewNamespacePermissionEvaluator returns an evaluator that evaluates the permissions of a subject in the specified namespace
func NewNamespacePermissionEvaluator(clusterID string, namespace string, roleStore roleStore.DataStore, bindingStore bindingStore.DataStore) k8srbac.EvaluatorForContext {
	return &namespacePermissionEvaluator{
		clusterID:     clusterID,
		namespace:     namespace,
		roleStore:     roleStore,
		bindingsStore: bindingStore,
	}
}

func (c *namespacePermissionEvaluator) ForSubject(ctx context.Context, subject *storage.Subject) k8srbac.PolicyRuleSet {
	roleBindings, roles := c.getBindingsAndRoles(ctx, subject)
	evaluator := k8srbac.NewEvaluator(roles, roleBindings)
	return evaluator.ForSubject(subject)
}

// IsClusterAdmin returns true if the subject has cluster admin. privs, false otherwise
func (c *namespacePermissionEvaluator) IsClusterAdmin(_ context.Context, _ *storage.Subject) bool {
	return false
}

// RolesForSubject returns the roles assigned to the subject based on the evaluator's bindings
func (c *namespacePermissionEvaluator) RolesForSubject(ctx context.Context, subject *storage.Subject) []*storage.K8SRole {
	clusterRoleBindings, roles := c.getBindingsAndRoles(ctx, subject)
	evaluator := k8srbac.NewEvaluator(roles, clusterRoleBindings)
	return evaluator.RolesForSubject(subject)
}

func (c *namespacePermissionEvaluator) getBindingsAndRoles(ctx context.Context, subject *storage.Subject) ([]*storage.K8SRoleBinding, []*storage.K8SRole) {
	q := search.NewQueryBuilder().
		AddExactMatches(search.ClusterID, c.clusterID).
		AddExactMatches(search.Namespace, c.namespace).
		AddBools(search.ClusterRole, false).
		AddExactMatches(search.SubjectName, subject.GetName()).
		AddExactMatches(search.SubjectKind, subject.GetKind().String()).
		ProtoQuery()
	rolebindings, err := c.bindingsStore.SearchRawRoleBindings(ctx, q)

	if err != nil {
		log.Errorf("error searching for rolebindings: %v", err)
		return nil, nil
	}

	roles := getRolesForBindings(ctx, c.roleStore, rolebindings)
	return rolebindings, roles
}
