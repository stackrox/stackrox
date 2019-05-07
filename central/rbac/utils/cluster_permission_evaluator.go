package utils

import (
	"context"

	roleStore "github.com/stackrox/rox/central/rbac/k8srole/datastore"
	bindingStore "github.com/stackrox/rox/central/rbac/k8srolebinding/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/k8srbac"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/search"
)

var (
	log = logging.LoggerForModule()
)

type clusterPermissionEvaluator struct {
	clusterID     string
	roleStore     roleStore.DataStore
	bindingsStore bindingStore.DataStore
}

// NewClusterPermissionEvaluator returns an evaluator that evaluates the permissions of a subject cluster wide.
func NewClusterPermissionEvaluator(clusterID string, roleStore roleStore.DataStore, bindingStore bindingStore.DataStore) k8srbac.Evaluator {
	return &clusterPermissionEvaluator{
		clusterID:     clusterID,
		roleStore:     roleStore,
		bindingsStore: bindingStore,
	}
}

func (c *clusterPermissionEvaluator) ForSubject(subject *storage.Subject) k8srbac.PolicyRuleSet {
	clusterRoleBindings, roles := c.getBindingsAndRoles()
	evaluator := k8srbac.NewEvaluator(roles, clusterRoleBindings)
	return evaluator.ForSubject(subject)
}

// IsClusterAdmin returns true if the subject has cluster admin. privs, false otherwise
func (c *clusterPermissionEvaluator) IsClusterAdmin(subject *storage.Subject) bool {
	clusterRoleBindings, roles := c.getBindingsAndRoles()
	evaluator := k8srbac.NewEvaluator(roles, clusterRoleBindings)
	return evaluator.IsClusterAdmin(subject)
}

// RolesForSubject returns the roles assigned to the subject based on the evaluator's bindings
func (c *clusterPermissionEvaluator) RolesForSubject(subject *storage.Subject) []*storage.K8SRole {
	clusterRoleBindings, roles := c.getBindingsAndRoles()
	evaluator := k8srbac.NewEvaluator(roles, clusterRoleBindings)
	return evaluator.RolesForSubject(subject)
}

func (c *clusterPermissionEvaluator) getBindingsAndRoles() ([]*storage.K8SRoleBinding, []*storage.K8SRole) {

	q := search.NewQueryBuilder().
		AddExactMatches(search.ClusterID, c.clusterID).
		AddBools(search.ClusterRole, true).ProtoQuery()
	clusterRoleBindings, err := c.bindingsStore.SearchRawRoleBindings(context.TODO(), q)

	if err != nil {
		log.Errorf("error searching for clusterrolebindings: %v", err)
		return nil, nil
	}

	roles := getRolesForBindings(c.roleStore, clusterRoleBindings)
	return clusterRoleBindings, roles
}
