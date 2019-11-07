package resolvers

import (
	"context"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/rox/central/metrics"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/k8srbac"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddQuery("k8sRoles(query: String): [K8SRole!]!"),
		schema.AddExtraResolver("K8SRole", `cluster: Cluster!`),
		schema.AddExtraResolver("K8SRole", `type: String!`),
		schema.AddExtraResolver("K8SRole", `verbs: [String!]!`),
		schema.AddExtraResolver("K8SRole", `resources: [String!]!`),
		schema.AddExtraResolver("K8SRole", `urls: [String!]!`),
		schema.AddExtraResolver("K8SRole", `subjectCount: Int!`),
		schema.AddExtraResolver("K8SRole", `subjects(query: String): [SubjectWithClusterID!]!`),
		schema.AddExtraResolver("K8SRole", `serviceAccountCount: Int!`),
		schema.AddExtraResolver("K8SRole", `serviceAccounts(query: String): [ServiceAccount!]!`),
		schema.AddExtraResolver("K8SRole", `roleNamespace: Namespace`),
	)
}

// Cluster returns a GraphQL resolver for the cluster of this role
func (resolver *k8SRoleResolver) Cluster(ctx context.Context) (*clusterResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.K8sRoles, "Cluster")

	if err := readClusters(ctx); err != nil {
		return nil, err
	}

	clusterID := graphql.ID(resolver.data.GetClusterId())
	return resolver.root.Cluster(ctx, struct{ graphql.ID }{clusterID})
}

// K8sRoles return k8s roles based on a query
func (resolver *Resolver) K8sRoles(ctx context.Context, arg rawQuery) ([]*k8SRoleResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "K8sRoles")

	if err := readK8sRoles(ctx); err != nil {
		return nil, err
	}
	query, err := arg.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}

	k8sRoles, err := resolver.K8sRoleStore.SearchRawRoles(ctx, query)
	if err != nil {
		return nil, err
	}

	var k8SRoleResolvers []*k8SRoleResolver
	for _, k8srole := range k8sRoles {
		k8SRoleResolvers = append(k8SRoleResolvers, &k8SRoleResolver{root: resolver, data: k8srole})
	}
	return k8SRoleResolvers, nil
}

func (resolver *k8SRoleResolver) Type(ctx context.Context) (string, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.K8sRoles, "Type")

	if err := readK8sRoles(ctx); err != nil {
		return "", err
	}

	if resolver.data.GetClusterRole() {
		return "ClusterRole", nil
	}
	return "Role", nil
}

// Verbs returns the set of verbs granted by a given k8s role
func (resolver *k8SRoleResolver) Verbs(ctx context.Context) ([]string, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.K8sRoles, "Verbs")

	if err := readK8sRoles(ctx); err != nil {
		return nil, err
	}

	return k8srbac.GetVerbsForRole(resolver.data).AsSlice(), nil

}

// Resources returns the set of resources that have been granted permissions to by a given k8s role
func (resolver *k8SRoleResolver) Resources(ctx context.Context) ([]string, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.K8sRoles, "Resources")

	if err := readK8sRoles(ctx); err != nil {
		return nil, err
	}
	return k8srbac.GetResourcesForRole(resolver.data).AsSlice(), nil

}

// NonResourceURLs returns the set of non resource urls granted permissions to by a given k8s role
func (resolver *k8SRoleResolver) Urls(ctx context.Context) ([]string, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.K8sRoles, "Urls")

	if err := readK8sRoles(ctx); err != nil {
		return nil, err
	}
	return k8srbac.GetNonResourceURLsForRole(resolver.data).AsSlice(), nil
}

// SubjectCount returns the number of subjects granted permissions to by a given k8s role
func (resolver *k8SRoleResolver) SubjectCount(ctx context.Context) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.K8sRoles, "SubjectCount")

	if err := readK8sSubjects(ctx); err != nil {
		return 0, err
	}

	subjects, err := resolver.getSubjects(ctx, rawQuery{})
	if err != nil {
		return 0, err
	}
	return int32(len(subjects)), nil
}

// Subjects returns the set of subjects granted permissions to by a given k8s role
func (resolver *k8SRoleResolver) Subjects(ctx context.Context, args rawQuery) ([]*subjectWithClusterIDResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.K8sRoles, "Subjects")

	if err := readK8sSubjects(ctx); err != nil {
		return nil, err
	}

	subjects, err := resolver.root.wrapSubjects(resolver.getSubjects(ctx, args))
	if err != nil {
		return nil, err
	}
	return wrapSubjects(resolver.data.GetClusterId(), resolver.data.GetClusterName(), subjects), nil
}

func (resolver *k8SRoleResolver) getSubjects(ctx context.Context, args rawQuery) ([]*storage.Subject, error) {
	subjects := make([]*storage.Subject, 0)

	filterQ, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return subjects, err
	}

	q := search.NewQueryBuilder().AddExactMatches(search.ClusterID, resolver.data.GetClusterId()).
		AddExactMatches(search.RoleID, resolver.data.GetId()).ProtoQuery()
	bindings, err := resolver.root.K8sRoleBindingStore.SearchRawRoleBindings(ctx, search.NewConjunctionQuery(q, filterQ))

	if err != nil {
		return subjects, err
	}
	return k8srbac.GetAllSubjects(bindings,
		storage.SubjectKind_USER, storage.SubjectKind_GROUP), nil
}

// ServiceAccounts returns the set of service accounts granted permissions to by a given k8s role
func (resolver *k8SRoleResolver) ServiceAccounts(ctx context.Context, args rawQuery) ([]*serviceAccountResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.K8sRoles, "ServiceAccounts")

	if err := readServiceAccounts(ctx); err != nil {
		return nil, err
	}

	return resolver.root.wrapServiceAccounts(resolver.getServiceAccounts(ctx, args))
}

// ServiceAccountCount returns the count of service accounts granted permissions to by a given k8s role
func (resolver *k8SRoleResolver) ServiceAccountCount(ctx context.Context) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.K8sRoles, "ServiceAccountCount")

	if err := readServiceAccounts(ctx); err != nil {
		return 0, err
	}
	serviceAccounts, err := resolver.getServiceAccounts(ctx, rawQuery{})
	if err != nil {
		return 0, err
	}
	return int32(len(serviceAccounts)), nil
}

func (resolver *k8SRoleResolver) getServiceAccounts(ctx context.Context, args rawQuery) ([]*storage.ServiceAccount, error) {
	filterQ, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}

	q := search.NewQueryBuilder().AddExactMatches(search.ClusterID, resolver.data.GetClusterId()).
		AddExactMatches(search.RoleID, resolver.data.GetId()).ProtoQuery()
	bindings, err := resolver.root.K8sRoleBindingStore.SearchRawRoleBindings(ctx, q)

	if err != nil {
		return nil, err
	}

	subjects := k8srbac.GetAllSubjects(bindings, storage.SubjectKind_SERVICE_ACCOUNT)
	accounts := make([]*storage.ServiceAccount, 0, len(subjects))
	for _, subject := range subjects {
		sa, err := resolver.convertSubjectToServiceAccount(ctx, resolver.data.GetClusterId(), subject, filterQ)
		if err != nil {
			log.Warnf("error converting subject to service account: %v", err)
			continue
		}
		if sa == nil {
			log.Warnf("service account: %s does not exist", subject.GetName())
			continue
		}
		accounts = append(accounts, sa)
	}

	return accounts, nil
}

// RoleNamespace returns the namespace of the k8s role
func (resolver *k8SRoleResolver) RoleNamespace(ctx context.Context) (*namespaceResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.K8sRoles, "RoleNamespace")

	role := resolver.data
	if role.GetNamespace() == "" {
		return nil, nil
	}
	return resolver.root.NamespaceByClusterIDAndName(ctx, clusterIDAndNameQuery{graphql.ID(role.GetClusterId()), role.GetNamespace()})
}

func (resolver *k8SRoleResolver) convertSubjectToServiceAccount(ctx context.Context, clusterID string, subject *storage.Subject, filterQ *v1.Query) (*storage.ServiceAccount, error) {
	q := search.NewQueryBuilder().AddExactMatches(search.ClusterID, clusterID).
		AddExactMatches(search.ServiceAccountName, subject.GetName()).ProtoQuery()

	serviceAccounts, err := resolver.root.ServiceAccountsDataStore.SearchRawServiceAccounts(ctx, search.NewConjunctionQuery(q, filterQ))
	if err != nil {
		return nil, err
	}
	if len(serviceAccounts) == 0 {
		return nil, nil
	}
	return serviceAccounts[0], nil
}
