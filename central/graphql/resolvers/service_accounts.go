package resolvers

import (
	"context"

	"github.com/graph-gophers/graphql-go"
	rbacUtils "github.com/stackrox/rox/central/rbac/utils"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/k8srbac"
	pkgRbacUtils "github.com/stackrox/rox/pkg/k8srbac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddQuery("serviceAccount(id: ID!): ServiceAccount"),
		schema.AddType("StringListEntry", []string{"key: String!", "values: [String!]!"}),
		schema.AddType("ScopedPermissions", []string{"scope: String!", "permissions: [StringListEntry!]!"}),
		schema.AddExtraResolver("ServiceAccount", `roles: [K8SRole!]!`),
		schema.AddExtraResolver("ServiceAccount", `scopedPermissions: [ScopedPermissions!]!`),
		schema.AddExtraResolver("ServiceAccount", `deployments: [Deployment!]!`),
		schema.AddExtraResolver("ServiceAccount", `saNamespace: Namespace!`),
		schema.AddExtraResolver("ServiceAccount", `clusterAdmin: Boolean!`),
	)
}

// ServiceAccount gets a service account by ID.
func (resolver *Resolver) ServiceAccount(ctx context.Context, args struct{ graphql.ID }) (*serviceAccountResolver, error) {
	if err := readServiceAccounts(ctx); err != nil {
		return nil, err
	}
	return resolver.wrapServiceAccount(resolver.ServiceAccountsDataStore.GetServiceAccount(ctx, string(args.ID)))
}

func (resolver *serviceAccountResolver) Roles(ctx context.Context) ([]*k8SRoleResolver, error) {
	if err := readK8sRoles(ctx); err != nil {
		return nil, err
	}
	if err := readK8sRoleBindings(ctx); err != nil {
		return nil, err
	}

	q := search.NewQueryBuilder().AddExactMatches(search.ClusterID, resolver.data.GetClusterId()).ProtoQuery()
	bindings, err := resolver.root.K8sRoleBindingStore.SearchRawRoleBindings(ctx, q)

	if err != nil {
		return nil, err
	}

	q = search.NewQueryBuilder().AddExactMatches(search.ClusterID, resolver.data.GetClusterId()).ProtoQuery()
	roles, err := resolver.root.K8sRoleStore.SearchRawRoles(ctx, q)

	if err != nil {
		return nil, err
	}

	subject := &storage.Subject{
		Name:      resolver.data.GetName(),
		Namespace: resolver.data.GetNamespace(),
		Kind:      storage.SubjectKind_SERVICE_ACCOUNT,
	}

	return resolver.root.wrapK8SRoles(k8srbac.NewEvaluator(roles, bindings).RolesForSubject(subject), nil)
}

func (resolver *serviceAccountResolver) Deployments(ctx context.Context) ([]*deploymentResolver, error) {
	if err := readDeployments(ctx); err != nil {
		return nil, err
	}

	q := search.NewQueryBuilder().AddExactMatches(search.ClusterID, resolver.data.GetClusterId()).
		AddExactMatches(search.ServiceAccountName, resolver.data.GetName()).ProtoQuery()
	deployments, err := resolver.root.DeploymentDataStore.SearchListDeployments(ctx, q)

	if err != nil {
		return nil, err
	}

	return resolver.root.wrapListDeployments(deployments, nil)
}

// Permission returns which scopes do the permissions for the service acc
func (resolver *serviceAccountResolver) ScopedPermissions(ctx context.Context) ([]*scopedPermissionsResolver, error) {
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

	subject := &storage.Subject{
		Name:      resolver.data.GetName(),
		Namespace: resolver.data.GetNamespace(),
		Kind:      storage.SubjectKind_SERVICE_ACCOUNT,
	}

	permissionScopeMap := make(map[string]map[string]set.StringSet)
	for scope, evaluator := range evaluators {
		permissions := evaluator.ForSubject(subject).GetPermissionMap()
		if len(permissions) != 0 {
			permissionScopeMap[scope] = permissions
		}
	}

	return wrapPermissions(permissionScopeMap), nil
}

// SaNamespace returns the namespace of the service account
func (resolver *serviceAccountResolver) SaNamespace(ctx context.Context) (*namespaceResolver, error) {
	sa := resolver.data
	r, err := resolver.root.NamespaceByClusterIDAndName(ctx, clusterIDAndNameQuery{graphql.ID(sa.GetClusterId()), sa.GetNamespace()})

	if err != nil {
		return resolver.root.wrapNamespace(r.data, false, err)
	}

	return resolver.root.wrapNamespace(r.data, true, err)
}

// ClusterAdmin returns if the service account is a cluster admin or not
func (resolver *serviceAccountResolver) ClusterAdmin(ctx context.Context) (bool, error) {
	sa := pkgRbacUtils.GetSubjectForServiceAccount(resolver.data)
	evaluator := resolver.getClusterEvaluator(ctx)

	return evaluator.IsClusterAdmin(sa), nil
}

func (resolver *serviceAccountResolver) getEvaluators(ctx context.Context) (map[string]k8srbac.Evaluator, error) {
	evaluators := make(map[string]k8srbac.Evaluator)
	saClusterID := resolver.data.GetClusterId()

	evaluators["Cluster"] =
		rbacUtils.NewClusterPermissionEvaluator(saClusterID,
			resolver.root.K8sRoleStore, resolver.root.K8sRoleBindingStore)

	namespaces, err := resolver.root.Namespaces(ctx)
	if err != nil {
		return evaluators, err
	}
	for _, namespace := range namespaces {
		namespaceName := namespace.data.GetMetadata().GetName()
		evaluators[namespaceName] = rbacUtils.NewNamespacePermissionEvaluator(saClusterID,
			namespaceName, resolver.root.K8sRoleStore, resolver.root.K8sRoleBindingStore)
	}

	return evaluators, nil
}

func (resolver *serviceAccountResolver) getClusterEvaluator(ctx context.Context) k8srbac.Evaluator {
	saClusterID := resolver.data.GetClusterId()

	return rbacUtils.NewClusterPermissionEvaluator(saClusterID,
		resolver.root.K8sRoleStore, resolver.root.K8sRoleBindingStore)
}
