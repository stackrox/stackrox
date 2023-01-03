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
func NewClusterPermissionEvaluator(clusterID string, roleStore roleStore.DataStore, bindingStore bindingStore.DataStore) k8srbac.EvaluatorForContext {
	return &clusterPermissionEvaluator{
		clusterID:     clusterID,
		roleStore:     roleStore,
		bindingsStore: bindingStore,
	}
}

func (c *clusterPermissionEvaluator) ForSubject(ctx context.Context, subject *storage.Subject) k8srbac.PolicyRuleSet {
	clusterRoleBindings, roles := c.getBindingsAndRoles(ctx, subject)
	evaluator := k8srbac.NewEvaluator(roles, clusterRoleBindings)
	return evaluator.ForSubject(subject)
}

// IsClusterAdmin returns true if the subject has cluster admin. privs, false otherwise
func (c *clusterPermissionEvaluator) IsClusterAdmin(ctx context.Context, subject *storage.Subject) bool {
	clusterRoleBindings, roles := c.getBindingsAndRoles(ctx, subject)
	evaluator := k8srbac.NewEvaluator(roles, clusterRoleBindings)
	return evaluator.IsClusterAdmin(subject)
}

// RolesForSubject returns the roles assigned to the subject based on the evaluator's bindings
func (c *clusterPermissionEvaluator) RolesForSubject(ctx context.Context, subject *storage.Subject) []*storage.K8SRole {
	clusterRoleBindings, roles := c.getBindingsAndRoles(ctx, subject)
	evaluator := k8srbac.NewEvaluator(roles, clusterRoleBindings)
	return evaluator.RolesForSubject(subject)
}

func (c *clusterPermissionEvaluator) getBindingsAndRoles(ctx context.Context, subject *storage.Subject) ([]*storage.K8SRoleBinding, []*storage.K8SRole) {
	q := search.NewQueryBuilder().
		AddExactMatches(search.ClusterID, c.clusterID).
		// Only evaluate bindings which have bind a cluster role _and_ have no namespace. Otherwise, we are evaluating
		// role bindings which bind a cluster role to a specific namespace and would mistakenly
		// see them as "cluster scoped".
		AddNullField(search.Namespace).
		AddBools(search.ClusterRole, true).
		AddExactMatches(search.SubjectName, subject.GetName()).
		AddExactMatches(search.SubjectKind, subject.GetKind().String()).
		ProtoQuery()
	clusterRoleBindings, err := c.bindingsStore.SearchRawRoleBindings(ctx, q)

	if err != nil {
		log.Errorf("error searching for clusterrolebindings: %v", err)
		return nil, nil
	}
	roles := getRolesForRoleBindings(ctx, c.roleStore, clusterRoleBindings, c.clusterID, "")
	return clusterRoleBindings, roles
}
