package resolvers

import (
	"context"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/metrics"
	rbacUtils "github.com/stackrox/rox/central/rbac/utils"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
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
		schema.AddExtraResolvers("ServiceAccount", []string{
			`k8sRoles(query: String, pagination: Pagination): [K8SRole!]!`,
			`k8sRoleCount(query: String): Int!`,
			`scopedPermissions: [ScopedPermissions!]!`,
			`deployments(query: String, pagination: Pagination): [Deployment!]!`,
			`deploymentCount(query: String): Int!`,
			`saNamespace: Namespace!`,
			`cluster: Cluster!`,
			`clusterAdmin: Boolean!`,
			`imagePullSecretCount: Int!`,
			`imagePullSecretObjects(query: String): [Secret!]!`,
		}),
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

// TODO
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

	query := search.NewQueryBuilder().AddExactMatches(search.SubjectKind, storage.SubjectKind_SERVICE_ACCOUNT.String()).
		AddExactMatches(search.ClusterID, resolver.data.GetClusterId()).
		AddExactMatches(search.Namespace, resolver.data.GetNamespace()).
		AddExactMatches(search.SubjectName, resolver.data.GetName()).
		AddSelectFields(search.NewQuerySelect(search.RoleID)).
		ProtoQuery()

	// we can query the K8sRole datastore
	count, err := resolver.root.K8sRoleBindingStore.Count(ctx, search.ConjunctionQuery(q, query))
	//count, err := resolver.root.K8sRoleStore.Count(ctx, search.ConjunctionQuery(q, query))
	if err != nil {
		return 0, err
	}

	return int32(count), nil

	//bindings, roles, err := resolver.getRolesAndBindings(ctx, q)
	//if err != nil {
	//	return 0, err
	//}

	//subject := k8srbac.GetSubjectForServiceAccount(resolver.data)
	//return int32(len(k8srbac.NewEvaluator(roles, bindings).RolesForSubject(subject))), nil
}

// TODO
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

	query := search.NewQueryBuilder().AddExactMatches(search.SubjectKind, storage.SubjectKind_SERVICE_ACCOUNT.String()).
		AddExactMatches(search.ClusterID, resolver.data.GetClusterId()).
		AddExactMatches(search.Namespace, resolver.data.GetNamespace()).
		AddExactMatches(search.SubjectName, resolver.data.GetName()).
		AddSelectFields(search.NewQuerySelect(search.RoleID)).
		ProtoQuery()

	results, err := resolver.root.K8sRoleBindingStore.Search(ctx, search.ConjunctionQuery(q, query))
	if err != nil {
		log.Error("Failed to resolve associated Role Bindings:", err)
		return nil, errors.Wrap(err, "Failed to resolve associated Role Bindings")
	}
	log.Error("osward -- sa is", resolver.data.GetName())
	log.Error("osward --", results)

	for _, result := range results {
		role, found, err := resolver.root.K8sRoleStore.GetRole(ctx, result.ID)
		if err != nil || !found {
			return nil, err
		}
		log.Error("osward -- role is", role.GetName())
	}

	return nil, errors.New("TODO - osward")
	//resultIDs := search.ResultsToIDs(results)

	//for _, result := range results {
	//	result.ID
	//}
	//resolver.root.K8sRoleStore. TODO get many?
	//resolver.root.wrapK8SRoles(resolver.root.K8sRoleStore)

	//pagination := q.Pagination
	//q.Pagination = nil

	//bindings, roles, err := resolver.getRolesAndBindings(ctx, q)
	//if err != nil {
	//	return nil, err
	//}
	//subject := k8srbac.GetSubjectForServiceAccount(resolver.data)
	//roleResolvers, err := resolver.root.wrapK8SRoles(k8srbac.NewEvaluator(roles, bindings).RolesForSubject(subject), nil)
	//if err != nil {
	//	return nil, err
	//}

	//return paginate(pagination, roleResolvers, nil)
}

// TODO
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

// TODO
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

	q := clusterIDAndNameQuery{
		ClusterID: graphql.ID(resolver.data.GetClusterId()),
		Name:      resolver.data.GetNamespace(),
	}

	return resolver.root.NamespaceByClusterIDAndName(ctx, q)
}

// Cluster returns the cluster of the service account
func (resolver *serviceAccountResolver) Cluster(ctx context.Context) (*clusterResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ServiceAccounts, "Cluster")
	if err := readClusters(ctx); err != nil {
		return nil, err
	}

	return resolver.root.wrapCluster(resolver.root.ClusterDataStore.GetCluster(ctx, resolver.data.GetClusterId()))
}

// TODO
// ClusterAdmin returns if the service account is a cluster admin or not
func (resolver *serviceAccountResolver) ClusterAdmin(ctx context.Context) (bool, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ServiceAccounts, "ClusterAdmin")
	sa := k8srbac.GetSubjectForServiceAccount(resolver.data)
	evaluator := resolver.getClusterEvaluator(ctx)

	return evaluator.IsClusterAdmin(ctx, sa), nil
}

// TODO
func (resolver *serviceAccountResolver) getEvaluators(ctx context.Context) (map[string]k8srbac.EvaluatorForContext, error) {
	evaluators := make(map[string]k8srbac.EvaluatorForContext)
	saClusterID := resolver.data.GetClusterId()

	evaluators["Cluster"] =
		rbacUtils.NewClusterPermissionEvaluator(saClusterID,
			resolver.root.K8sRoleStore, resolver.root.K8sRoleBindingStore)

	if env.PostgresDatastoreEnabled.BooleanSetting() {
		ctx = scoped.Context(ctx, scoped.Scope{
			Level: v1.SearchCategory_CLUSTERS,
			ID:    saClusterID,
		})
	}
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

// TODO
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
