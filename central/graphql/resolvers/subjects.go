package resolvers

import (
	"context"

	"github.com/pkg/errors"
	rbacUtils "github.com/stackrox/rox/central/rbac/utils"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/k8srbac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddExtraResolver("SubjectWithClusterID", `name: String!`),
		schema.AddExtraResolver("SubjectWithClusterID", `namespace: String!`),
		schema.AddExtraResolver("SubjectWithClusterID", `type: String!`),
		schema.AddExtraResolver("SubjectWithClusterID", `type: String!`),
		schema.AddExtraResolver("SubjectWithClusterID", `roles: [K8SRole!]!`),
		schema.AddExtraResolver("SubjectWithClusterID", `scopedPermissions: [ScopedPermissions!]!`),
	)
}

func (resolver *subjectWithClusterIDResolver) Name(ctx context.Context) (string, error) {
	if err := readK8sSubjects(ctx); err != nil {
		return "", err
	}

	return resolver.subject.Name(ctx), nil
}

func (resolver *subjectWithClusterIDResolver) Namespace(ctx context.Context) (string, error) {
	if err := readK8sSubjects(ctx); err != nil {
		return "", err
	}

	return resolver.subject.Namespace(ctx), nil
}

func (resolver *subjectWithClusterIDResolver) Type(ctx context.Context) (string, error) {
	if err := readK8sSubjects(ctx); err != nil {
		return "", err
	}

	subject := resolver.subject.data
	switch subject.GetKind() {
	case storage.SubjectKind_USER:
		return "User", nil
	case storage.SubjectKind_GROUP:
		return "Group", nil
	default:
		return "", errors.Errorf("invalid subject type")
	}
}

func (resolver *subjectWithClusterIDResolver) Roles(ctx context.Context) ([]*k8SRoleResolver, error) {
	if err := readK8sRoles(ctx); err != nil {
		return nil, err
	}
	if err := readK8sRoleBindings(ctx); err != nil {
		return nil, err
	}

	q := search.NewQueryBuilder().AddExactMatches(search.ClusterID, resolver.clusterID).ProtoQuery()
	bindings, err := resolver.subject.root.K8sRoleBindingStore.SearchRawRoleBindings(ctx, q)

	if err != nil {
		return nil, err
	}

	q = search.NewQueryBuilder().AddExactMatches(search.ClusterID, resolver.clusterID).ProtoQuery()
	roles, err := resolver.subject.root.K8sRoleStore.SearchRawRoles(ctx, q)

	if err != nil {
		return nil, err
	}

	return resolver.subject.root.wrapK8SRoles(k8srbac.NewEvaluator(roles, bindings).RolesForSubject(resolver.subject.data), nil)

}

// Permission returns which scopes do the permissions for the subject
func (resolver *subjectWithClusterIDResolver) ScopedPermissions(ctx context.Context) ([]*scopedPermissionsResolver, error) {
	if err := readK8sRoles(ctx); err != nil {
		return nil, err
	}

	if err := readK8sRoleBindings(ctx); err != nil {
		return nil, err
	}

	evaluators, err := resolver.getEvaluators(ctx)
	if err != nil {
		return nil, err
	}

	permissionScopeMap := make(map[string]map[string]set.StringSet)
	for scope, evaluator := range evaluators {
		permissions := evaluator.ForSubject(resolver.subject.data).GetPermissionMap()
		if len(permissions) != 0 {
			permissionScopeMap[scope] = permissions
		}
	}

	return wrapPermissions(permissionScopeMap), nil
}

func (resolver *subjectWithClusterIDResolver) getEvaluators(ctx context.Context) (map[string]k8srbac.Evaluator, error) {
	evaluators := make(map[string]k8srbac.Evaluator)
	clusterID := resolver.clusterID
	rootResolver := resolver.subject.root

	evaluators["Cluster"] =
		rbacUtils.NewClusterPermissionEvaluator(clusterID,
			rootResolver.K8sRoleStore, rootResolver.K8sRoleBindingStore)

	namespaces, err := rootResolver.Namespaces(ctx)
	if err != nil {
		return evaluators, err
	}
	for _, namespace := range namespaces {
		namespaceName := namespace.data.GetMetadata().GetName()
		evaluators[namespaceName] = rbacUtils.NewNamespacePermissionEvaluator(clusterID,
			namespaceName, rootResolver.K8sRoleStore, rootResolver.K8sRoleBindingStore)
	}

	return evaluators, nil
}
