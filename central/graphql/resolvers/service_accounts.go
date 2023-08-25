package resolvers

import (
	"context"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/rox/central/metrics"
	rbacUtils "github.com/stackrox/rox/central/rbac/utils"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/k8srbac"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/scoped"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddQuery("serviceAccount(id: ID!): ServiceAccount"),
		schema.AddQuery("serviceAccounts(query: String, pagination: Pagination): [ServiceAccount!]!"),
		schema.AddQuery("serviceAccountCount(query: String): Int!"),
		schema.AddType("StringListEntry", []string{"key: String!", "values: [String!]!"}),
		schema.AddType("ScopedPermissions", []string{"scope: String!", "permissions: [StringListEntry!]!"}),
		schema.AddExtraResolver("ServiceAccount", `k8sRoles(query: String, pagination: Pagination): [K8SRole!]!`),
		schema.AddExtraResolver("ServiceAccount", `k8sRoleCount(query: String): Int!`),
		schema.AddExtraResolver("ServiceAccount", `scopedPermissions: [ScopedPermissions!]!`),
		schema.AddExtraResolver("ServiceAccount", `deployments(query: String, pagination: Pagination): [Deployment!]!`),
		schema.AddExtraResolver("ServiceAccount", `deploymentCount(query: String): Int!`),
		schema.AddExtraResolver("ServiceAccount", `saNamespace: Namespace!`),
		schema.AddExtraResolver("ServiceAccount", `cluster: Cluster!`),
		schema.AddExtraResolver("ServiceAccount", `clusterAdmin: Boolean!`),
		schema.AddExtraResolver("ServiceAccount", `imagePullSecretCount: Int!`),
		schema.AddExtraResolver("ServiceAccount", `imagePullSecretObjects(query: String): [Secret!]!`),
	)
}

// ServiceAccount gets a service account by ID.
func (resolver *Resolver) ServiceAccount(ctx context.Context, args struct{ graphql.ID }) (*serviceAccountResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "ServiceAccount")
	if err := readServiceAccounts(ctx); err != nil {
		return nil, err
	}
	return resolver.wrapServiceAccount(resolver.ServiceAccountsDataStore.GetServiceAccount(ctx, string(args.ID)))
}

// ServiceAccounts gets service accounts based on a query
func (resolver *Resolver) ServiceAccounts(ctx context.Context, args PaginatedQuery) ([]*serviceAccountResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "ServiceAccounts")
	if err := readServiceAccounts(ctx); err != nil {
		return nil, err
	}

	query, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}

	return resolver.wrapServiceAccounts(resolver.ServiceAccountsDataStore.SearchRawServiceAccounts(ctx, query))
}

// ServiceAccountCount returns count of all service accounts across infrastructure
func (resolver *Resolver) ServiceAccountCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "ServiceAccountCount")
	if err := readServiceAccounts(ctx); err != nil {
		return 0, err
	}
	query, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return 0, err
	}
	count, err := resolver.ServiceAccountsDataStore.Count(ctx, query)
	if err != nil {
		return 0, err
	}
	return int32(count), nil
}

func (resolver *serviceAccountResolver) K8sRoleCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ServiceAccounts, "RoleCount")
	if err := readK8sRoles(ctx); err != nil {
		return 0, err
	}
	if err := readK8sRoleBindings(ctx); err != nil {
		return 0, err
	}

	q, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return 0, err
	}

	bindings, roles, err := resolver.getRolesAndBindings(ctx, q)
	if err != nil {
		return 0, err
	}
	subject := k8srbac.GetSubjectForServiceAccount(resolver.data)
	return int32(len(k8srbac.NewEvaluator(roles, bindings).RolesForSubject(subject))), nil
}

func (resolver *serviceAccountResolver) K8sRoles(ctx context.Context, args PaginatedQuery) ([]*k8SRoleResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ServiceAccounts, "Roles")
	if err := readK8sRoles(ctx); err != nil {
		return nil, err
	}
	if err := readK8sRoleBindings(ctx); err != nil {
		return nil, err
	}

	q, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}
	pagination := q.Pagination
	q.Pagination = nil

	bindings, roles, err := resolver.getRolesAndBindings(ctx, q)
	if err != nil {
		return nil, err
	}
	subject := k8srbac.GetSubjectForServiceAccount(resolver.data)
	roleResolvers, err := resolver.root.wrapK8SRoles(k8srbac.NewEvaluator(roles, bindings).RolesForSubject(subject), nil)
	if err != nil {
		return nil, err
	}

	return paginate(pagination, roleResolvers, nil)
}

func (resolver *serviceAccountResolver) getRolesAndBindings(ctx context.Context, passedQuery *v1.Query) ([]*storage.K8SRoleBinding, []*storage.K8SRole, error) {
	bindingQuery := search.NewQueryBuilder().AddExactMatches(search.ClusterID, resolver.data.GetClusterId()).ProtoQuery()
	bindings, err := resolver.root.K8sRoleBindingStore.SearchRawRoleBindings(ctx, bindingQuery)
	if err != nil {
		return nil, nil, err
	}

	bindingQuery = search.NewQueryBuilder().AddExactMatches(search.ClusterID, resolver.data.GetClusterId()).ProtoQuery()
	roles, err := resolver.root.K8sRoleStore.SearchRawRoles(ctx, search.ConjunctionQuery(passedQuery, bindingQuery))
	if err != nil {
		return nil, nil, err
	}
	return bindings, roles, nil
}

func (resolver *serviceAccountResolver) Deployments(ctx context.Context, args PaginatedQuery) ([]*deploymentResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ServiceAccounts, "Deployments")
	if err := readDeployments(ctx); err != nil {
		return nil, err
	}
	q, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}
	scopedQuery := search.NewQueryBuilder().AddExactMatches(search.ClusterID, resolver.data.GetClusterId()).
		AddExactMatches(search.Namespace, resolver.data.GetNamespace()).
		AddExactMatches(search.ServiceAccountName, resolver.data.GetName()).ProtoQuery()

	return resolver.root.wrapDeployments(resolver.root.DeploymentDataStore.SearchRawDeployments(ctx, search.ConjunctionQuery(scopedQuery, q)))
}

func (resolver *serviceAccountResolver) DeploymentCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ServiceAccounts, "DeploymentCount")
	if err := readDeployments(ctx); err != nil {
		return 0, err
	}

	q, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return 0, err
	}

	scopedQuery := search.NewQueryBuilder().AddExactMatches(search.ClusterID, resolver.data.GetClusterId()).
		AddExactMatches(search.Namespace, resolver.data.GetNamespace()).
		AddExactMatches(search.ServiceAccountName, resolver.data.GetName()).ProtoQuery()

	count, err := resolver.root.DeploymentDataStore.Count(ctx, search.ConjunctionQuery(scopedQuery, q))
	if err != nil {
		return 0, err
	}
	return int32(count), nil
}

// ScopedPermissions returns which scopes do the permissions for the service acc
func (resolver *serviceAccountResolver) ScopedPermissions(ctx context.Context) ([]*scopedPermissionsResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ServiceAccounts, "ScopedPermissions")
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

	subject := k8srbac.GetSubjectForServiceAccount(resolver.data)
	permissionScopeMap := make(map[string]map[string]set.StringSet)
	for scope, evaluator := range evaluators {
		permissions := evaluator.ForSubject(ctx, subject).GetPermissionMap()
		if len(permissions) != 0 {
			permissionScopeMap[scope] = permissions
		}
	}

	return wrapPermissions(permissionScopeMap), nil
}

// SaNamespace returns the namespace of the service account
func (resolver *serviceAccountResolver) SaNamespace(ctx context.Context) (*namespaceResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ServiceAccounts, "SaNamespace")
	sa := resolver.data
	return resolver.root.NamespaceByClusterIDAndName(ctx, clusterIDAndNameQuery{graphql.ID(sa.GetClusterId()), sa.GetNamespace()})
}

// Cluster returns the cluster of the service account
func (resolver *serviceAccountResolver) Cluster(ctx context.Context) (*clusterResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ServiceAccounts, "Cluster")
	if err := readClusters(ctx); err != nil {
		return nil, err
	}
	return resolver.root.wrapCluster(resolver.root.ClusterDataStore.GetCluster(ctx, resolver.data.GetClusterId()))
}

// ClusterAdmin returns if the service account is a cluster admin or not
func (resolver *serviceAccountResolver) ClusterAdmin(ctx context.Context) (bool, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ServiceAccounts, "ClusterAdmin")
	sa := k8srbac.GetSubjectForServiceAccount(resolver.data)
	evaluator := resolver.getClusterEvaluator(ctx)

	return evaluator.IsClusterAdmin(ctx, sa), nil
}

func (resolver *serviceAccountResolver) getEvaluators(ctx context.Context) (map[string]k8srbac.EvaluatorForContext, error) {
	evaluators := make(map[string]k8srbac.EvaluatorForContext)
	saClusterID := resolver.data.GetClusterId()

	evaluators["Cluster"] =
		rbacUtils.NewClusterPermissionEvaluator(saClusterID,
			resolver.root.K8sRoleStore, resolver.root.K8sRoleBindingStore)

	ctx = scoped.Context(ctx, scoped.Scope{
		Level: v1.SearchCategory_CLUSTERS,
		ID:    saClusterID,
	})

	namespaces, err := resolver.root.Namespaces(ctx, PaginatedQuery{})

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

func (resolver *serviceAccountResolver) getClusterEvaluator(_ context.Context) k8srbac.EvaluatorForContext {
	saClusterID := resolver.data.GetClusterId()

	return rbacUtils.NewClusterPermissionEvaluator(saClusterID,
		resolver.root.K8sRoleStore, resolver.root.K8sRoleBindingStore)
}

func (resolver *serviceAccountResolver) ImagePullSecretCount(ctx context.Context) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ServiceAccounts, "ImagePullSecretCount")
	if err := readSecrets(ctx); err != nil {
		return 0, err
	}
	return int32(len(resolver.data.ImagePullSecrets)), nil
}

func (resolver *serviceAccountResolver) ImagePullSecretObjects(ctx context.Context, args RawQuery) ([]*secretResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ServiceAccounts, "ImagePullSecretObjects")
	if err := readSecrets(ctx); err != nil {
		return nil, err
	}

	passedQuery, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}

	secretNames := resolver.data.GetImagePullSecrets()
	if len(secretNames) == 0 {
		return []*secretResolver{}, nil
	}
	q := search.NewQueryBuilder().AddExactMatches(search.SecretName, secretNames...).
		AddExactMatches(search.ClusterID, resolver.data.GetClusterId()).
		AddExactMatches(search.Namespace, resolver.data.GetNamespace()).ProtoQuery()

	q = search.ConjunctionQuery(passedQuery, q)
	return resolver.root.wrapSecrets(resolver.root.SecretsDataStore.SearchRawSecrets(ctx, q))
}
