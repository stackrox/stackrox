package utils

import (
	"context"

	roleStore "github.com/stackrox/rox/central/rbac/k8srole/datastore"
	bindingStore "github.com/stackrox/rox/central/rbac/k8srolebinding/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/k8srbac"
	"github.com/stackrox/rox/pkg/search"
)

type namespacePermissionEvaluator struct {
	clusterID     string
	namespace     string
	roleStore     roleStore.DataStore
	bindingsStore bindingStore.DataStore
}

// NewNamespacePermissionEvaluator returns an evaluator that evaluates the permissions of a subject in the specified namespace
func NewNamespacePermissionEvaluator(clusterID string, namespace string, roleStore roleStore.DataStore, bindingStore bindingStore.DataStore) k8srbac.Evaluator {
	return &namespacePermissionEvaluator{
		clusterID:     clusterID,
		namespace:     namespace,
		roleStore:     roleStore,
		bindingsStore: bindingStore,
	}
}

func (c *namespacePermissionEvaluator) ForSubject(subject *storage.Subject) k8srbac.PolicyRuleSet {
	roleBindings, roles := c.getBindingsAndRoles()
	evaluator := k8srbac.NewEvaluator(roles, roleBindings)
	return evaluator.ForSubject(subject)
}

// IsClusterAdmin returns true if the subject has cluster admin. privs, false otherwise
func (c *namespacePermissionEvaluator) IsClusterAdmin(subject *storage.Subject) bool {
	return false
}

// RolesForSubject returns the roles assigned to the subject based on the evaluator's bindings
func (c *namespacePermissionEvaluator) RolesForSubject(subject *storage.Subject) []*storage.K8SRole {
	clusterRoleBindings, roles := c.getBindingsAndRoles()
	evaluator := k8srbac.NewEvaluator(roles, clusterRoleBindings)
	return evaluator.RolesForSubject(subject)
}

func (c *namespacePermissionEvaluator) getBindingsAndRoles() ([]*storage.K8SRoleBinding, []*storage.K8SRole) {
	q := search.NewQueryBuilder().
		AddExactMatches(search.ClusterID, c.clusterID).
		AddExactMatches(search.Namespace, c.namespace).
		AddBools(search.ClusterRole, false).ProtoQuery()
	rolebindings, err := c.bindingsStore.SearchRawRoleBindings(context.TODO(), q)

	if err != nil {
		log.Errorf("error searching for rolebindings: %v", err)
		return nil, nil
	}

	roles := getRolesForBindings(c.roleStore, rolebindings)
	return rolebindings, roles
}
