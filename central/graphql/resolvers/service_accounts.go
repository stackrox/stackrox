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
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddQuery("serviceAccount(id: ID!): ServiceAccount"),
		schema.AddQuery("serviceAccounts(query: String, pagination: Pagination): [ServiceAccount!]!"),
		schema.AddType("StringListEntry", []string{"key: String!", "values: [String!]!"}),
		schema.AddType("ScopedPermissions", []string{"scope: String!", "permissions: [StringListEntry!]!"}),
		schema.AddExtraResolver("ServiceAccount", `roles(query: String): [K8SRole!]!`),
		schema.AddExtraResolver("ServiceAccount", `roleCount:Int!`),
		schema.AddExtraResolver("ServiceAccount", `scopedPermissions: [ScopedPermissions!]!`),
		schema.AddExtraResolver("ServiceAccount", `deployments(query: String): [Deployment!]!`),
		schema.AddExtraResolver("ServiceAccount", `deploymentCount: Int!`),
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
func (resolver *Resolver) ServiceAccounts(ctx context.Context, args paginatedQuery) ([]*serviceAccountResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "ServiceAccounts")
	if err := readServiceAccounts(ctx); err != nil {
		return nil, err
	}

	query, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}

	resolvers, err := paginationWrapper{
		pv: query.Pagination,
	}.paginate(resolver.wrapServiceAccounts(resolver.ServiceAccountsDataStore.SearchRawServiceAccounts(ctx, query)))
	return resolvers.([]*serviceAccountResolver), err
}

func (resolver *serviceAccountResolver) RoleCount(ctx context.Context) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ServiceAccounts, "RoleCount")
	if err := readK8sRoles(ctx); err != nil {
		return 0, err
	}
	if err := readK8sRoleBindings(ctx); err != nil {
		return 0, err
	}

	bindings, roles, err := resolver.getRolesAndBindings(ctx, rawQuery{})
	if err != nil {
		return 0, err
	}
	subject := &storage.Subject{
		Name:      resolver.data.GetName(),
		Namespace: resolver.data.GetNamespace(),
		Kind:      storage.SubjectKind_SERVICE_ACCOUNT,
	}

	return int32(len(k8srbac.NewEvaluator(roles, bindings).RolesForSubject(subject))), nil
}

func (resolver *serviceAccountResolver) Roles(ctx context.Context, args rawQuery) ([]*k8SRoleResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ServiceAccounts, "Roles")
	if err := readK8sRoles(ctx); err != nil {
		return nil, err
	}
	if err := readK8sRoleBindings(ctx); err != nil {
		return nil, err
	}

	bindings, roles, err := resolver.getRolesAndBindings(ctx, args)
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

func (resolver *serviceAccountResolver) getRolesAndBindings(ctx context.Context, args rawQuery) ([]*storage.K8SRoleBinding, []*storage.K8SRole, error) {
	q := search.NewQueryBuilder().AddExactMatches(search.ClusterID, resolver.data.GetClusterId()).ProtoQuery()
	bindings, err := resolver.root.K8sRoleBindingStore.SearchRawRoleBindings(ctx, q)
	if err != nil {
		return nil, nil, err
	}

	q = search.NewQueryBuilder().AddExactMatches(search.ClusterID, resolver.data.GetClusterId()).ProtoQuery()
	q, err = resolver.getConjunctionQuery(args, q)
	if err != nil {
		return nil, nil, err
	}

	roles, err := resolver.root.K8sRoleStore.SearchRawRoles(ctx, q)
	if err != nil {
		return nil, nil, err
	}
	return bindings, roles, nil
}

func (resolver *serviceAccountResolver) Deployments(ctx context.Context, args rawQuery) ([]*deploymentResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ServiceAccounts, "Deployments")
	if err := readDeployments(ctx); err != nil {
		return nil, err
	}
	q := search.NewQueryBuilder().AddExactMatches(search.ClusterID, resolver.data.GetClusterId()).
		AddExactMatches(search.Namespace, resolver.data.GetNamespace()).
		AddExactMatches(search.ServiceAccountName, resolver.data.GetName()).ProtoQuery()

	q, err := resolver.getConjunctionQuery(args, q)
	if err != nil {
		return nil, err
	}
	return resolver.root.wrapDeployments(resolver.root.DeploymentDataStore.SearchRawDeployments(ctx, q))
}

func (resolver *serviceAccountResolver) DeploymentCount(ctx context.Context) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ServiceAccounts, "DeploymentCount")
	if err := readDeployments(ctx); err != nil {
		return 0, err
	}
	q := search.NewQueryBuilder().AddExactMatches(search.ClusterID, resolver.data.GetClusterId()).
		AddExactMatches(search.Namespace, resolver.data.GetNamespace()).
		AddExactMatches(search.ServiceAccountName, resolver.data.GetName()).ProtoQuery()

	results, err := resolver.root.DeploymentDataStore.Search(ctx, q)
	if err != nil {
		return 0, err
	}
	return int32(len(results)), nil
}

// Permission returns which scopes do the permissions for the service acc
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

	subject := &storage.Subject{
		Name:      resolver.data.GetName(),
		Namespace: resolver.data.GetNamespace(),
		Kind:      storage.SubjectKind_SERVICE_ACCOUNT,
	}

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

	namespaces, err := resolver.root.Namespaces(ctx, paginatedQuery{})
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

func (resolver *serviceAccountResolver) getClusterEvaluator(ctx context.Context) k8srbac.EvaluatorForContext {
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

func (resolver *serviceAccountResolver) ImagePullSecretObjects(ctx context.Context, args rawQuery) ([]*secretResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ServiceAccounts, "ImagePullSecretObjects")
	if err := readSecrets(ctx); err != nil {
		return nil, err
	}
	secretNames := resolver.data.GetImagePullSecrets()
	if len(secretNames) == 0 {
		return []*secretResolver{}, nil
	}
	q := search.NewQueryBuilder().AddExactMatches(search.SecretName, secretNames...).
		AddExactMatches(search.ClusterID, resolver.data.GetClusterId()).
		AddExactMatches(search.Namespace, resolver.data.GetNamespace()).ProtoQuery()
	q, err := resolver.getConjunctionQuery(args, q)
	if err != nil {
		return nil, err
	}
	return resolver.root.wrapSecrets(resolver.root.SecretsDataStore.SearchRawSecrets(ctx, q))
}

func (resolver *serviceAccountResolver) getConjunctionQuery(args rawQuery, q2 *v1.Query) (*v1.Query, error) {
	if args.Query == nil {
		return q2, nil
	}
	q1, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}
	return search.NewConjunctionQuery(q1, q2), nil
}
