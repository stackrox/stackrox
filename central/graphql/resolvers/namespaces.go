package resolvers

import (
	"context"

	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/rox/central/namespace"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/k8srbac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddQuery("namespaces(query: String): [Namespace!]!"),
		schema.AddQuery("namespace(id: ID!): Namespace"),
		schema.AddQuery("namespaceByClusterIDAndName(clusterID: ID!, name: String!): Namespace"),
		schema.AddExtraResolver("Namespace", "complianceResults(query: String): [ControlResult!]!"),
		schema.AddExtraResolver("Namespace", `subjectCount: Int!`),
		schema.AddExtraResolver("Namespace", `serviceAccountCount: Int!`),
		schema.AddExtraResolver("Namespace", `k8sroleCount: Int!`),
		schema.AddExtraResolver("Namespace", `policyCount: Int!`),
		schema.AddExtraResolver("Namespace", `policyStatus: Boolean!`),
		schema.AddExtraResolver("Namespace", `images: [Image!]!`),
		schema.AddExtraResolver("Namespace", `imageCount: Int!`),
	)
}

// Namespace returns a GraphQL resolver for the given namespace.
func (resolver *Resolver) Namespace(ctx context.Context, args struct{ graphql.ID }) (*namespaceResolver, error) {
	if err := readNamespaces(ctx); err != nil {
		return nil, err
	}
	return resolver.wrapNamespace(namespace.ResolveByID(ctx, string(args.ID), resolver.NamespaceDataStore, resolver.DeploymentDataStore, resolver.SecretsDataStore, resolver.NetworkPoliciesStore))
}

// Namespaces returns GraphQL resolvers for all namespaces based on an optional query.
func (resolver *Resolver) Namespaces(ctx context.Context, args rawQuery) ([]*namespaceResolver, error) {
	if err := readNamespaces(ctx); err != nil {
		return nil, err
	}
	query, err := args.AsV1Query()
	if err != nil {
		return nil, err
	}
	if query == nil {
		return resolver.wrapNamespaces(namespace.ResolveAll(ctx, resolver.NamespaceDataStore, resolver.DeploymentDataStore, resolver.SecretsDataStore, resolver.NetworkPoliciesStore))
	}
	return resolver.wrapNamespaces(namespace.ResolveByQuery(ctx, query, resolver.NamespaceDataStore, resolver.DeploymentDataStore, resolver.SecretsDataStore, resolver.NetworkPoliciesStore))
}

type clusterIDAndNameQuery struct {
	ClusterID graphql.ID
	Name      string
}

// NamespaceByClusterIDAndName returns a GraphQL resolver for the (unique) namespace specified by this query.
func (resolver *Resolver) NamespaceByClusterIDAndName(ctx context.Context, args clusterIDAndNameQuery) (*namespaceResolver, error) {
	if err := readNamespaces(ctx); err != nil {
		return nil, err
	}
	return resolver.wrapNamespace(namespace.ResolveByClusterIDAndName(ctx, string(args.ClusterID), args.Name, resolver.NamespaceDataStore, resolver.DeploymentDataStore, resolver.SecretsDataStore, resolver.NetworkPoliciesStore))
}

func (resolver *namespaceResolver) ComplianceResults(ctx context.Context, args rawQuery) ([]*controlResultResolver, error) {
	if err := readCompliance(ctx); err != nil {
		return nil, err
	}

	runResults, err := resolver.root.ComplianceAggregator.GetResultsWithEvidence(ctx, args.String())
	if err != nil {
		return nil, err
	}
	output := newBulkControlResults()
	nsID := resolver.data.GetMetadata().GetId()
	output.addDeploymentData(resolver.root, runResults, func(d *storage.Deployment, _ *v1.ComplianceControl) bool {
		return d.GetNamespaceId() == nsID
	})

	return *output, nil
}

// SubjectCount returns the count of Subjects which have any permission on this cluster namespace
func (resolver *namespaceResolver) SubjectCount(ctx context.Context) (int32, error) {
	if err := readK8sSubjects(ctx); err != nil {
		return 0, err
	}
	if err := readK8sRoleBindings(ctx); err != nil {
		return 0, err
	}
	q := search.NewQueryBuilder().AddExactMatches(search.ClusterID, resolver.data.GetMetadata().GetClusterId()).
		AddExactMatches(search.Namespace, resolver.data.GetMetadata().GetName()).ProtoQuery()
	bindings, err := resolver.root.K8sRoleBindingStore.SearchRawRoleBindings(ctx, q)
	if err != nil {
		return 0, err
	}
	subjects := k8srbac.GetAllSubjects(bindings, storage.SubjectKind_USER, storage.SubjectKind_GROUP)
	return int32(len(subjects)), nil
}

// ServiceAccountCount returns the count of ServiceAccounts which have any permission on this cluster namespace
func (resolver *namespaceResolver) ServiceAccountCount(ctx context.Context) (int32, error) {
	if err := readServiceAccounts(ctx); err != nil {
		return 0, err
	}
	q := search.NewQueryBuilder().AddExactMatches(search.ClusterID, resolver.data.GetMetadata().GetClusterId()).
		AddExactMatches(search.Namespace, resolver.data.GetMetadata().GetName()).ProtoQuery()
	results, err := resolver.root.ServiceAccountsDataStore.Search(ctx, q)
	if err != nil {
		return 0, err
	}
	return int32(len(results)), nil
}

// K8sRoleCount returns count of K8s roles in this cluster namespace
func (resolver *namespaceResolver) K8sRoleCount(ctx context.Context) (int32, error) {
	if err := readK8sRoles(ctx); err != nil {
		return 0, err
	}
	q := search.NewQueryBuilder().AddExactMatches(search.ClusterID, resolver.data.GetMetadata().GetClusterId()).
		AddExactMatches(search.Namespace, resolver.data.GetMetadata().GetName()).ProtoQuery()
	results, err := resolver.root.K8sRoleStore.Search(ctx, q)
	if err != nil {
		return 0, err
	}
	return int32(len(results)), nil
}

func (resolver *namespaceResolver) Images(ctx context.Context) ([]*imageResolver, error) {
	if err := readNamespaces(ctx); err != nil {
		return nil, err
	}
	q := search.NewQueryBuilder().AddExactMatches(search.ClusterID, resolver.data.GetMetadata().GetClusterId()).
		AddExactMatches(search.Namespace, resolver.data.Metadata.GetName()).ProtoQuery()
	return resolver.root.wrapListImages(resolver.root.ImageDataStore.SearchListImages(ctx, q))
}

func (resolver *namespaceResolver) ImageCount(ctx context.Context) (int32, error) {
	if err := readNamespaces(ctx); err != nil {
		return 0, err
	}
	q := search.NewQueryBuilder().AddExactMatches(search.ClusterID, resolver.data.GetMetadata().GetClusterId()).
		AddExactMatches(search.Namespace, resolver.data.Metadata.GetName()).ProtoQuery()
	results, err := resolver.root.ImageDataStore.Search(ctx, q)
	if err != nil {
		return 0, err
	}
	return int32(len(results)), nil
}

func (resolver *namespaceResolver) Policies(ctx context.Context, clusterID string) ([]*storage.Policy, error) {
	if err := readPolicies(ctx); err != nil {
		return nil, err
	}
	policies, err := resolver.root.PolicyDataStore.GetPolicies(ctx)
	if err != nil {
		return nil, err
	}
	var filteredPolicies []*storage.Policy
	for _, policy := range policies {
		if resolver.policyAppliesToNamespace(ctx, policy, clusterID) {
			filteredPolicies = append(filteredPolicies, policy)
		}
	}
	return filteredPolicies, nil
}

// K8sRoleCount returns count of K8s roles in this cluster
func (resolver *namespaceResolver) PolicyCount(ctx context.Context) (int32, error) {
	policies, err := resolver.Policies(ctx, resolver.data.GetMetadata().GetClusterId())
	if err != nil {
		return 0, err
	}
	return int32(len(policies)), nil
}

func (resolver *namespaceResolver) policyAppliesToNamespace(ctx context.Context, policy *storage.Policy, clusterID string) bool {
	// Global Policy
	if len(policy.Scope) == 0 {
		return true
	}
	// Clustered or namespaced scope policy, evaluate all scopes
	for _, scope := range policy.Scope {
		if scope.GetCluster() != "" {
			if scope.GetCluster() == clusterID &&
				(scope.GetNamespace() == "" || scope.GetNamespace() == resolver.data.Metadata.GetName()) {
				return true
			}
		} else if scope.GetNamespace() != "" {
			if scope.GetNamespace() == resolver.data.GetMetadata().GetName() {
				return true
			}
		} else {
			return true
		}
	}
	return false
}

// PolicyStatus returns true if there is no policy violation for this cluster
func (resolver *namespaceResolver) PolicyStatus(ctx context.Context) (bool, error) {
	if err := readAlerts(ctx); err != nil {
		return false, err
	}
	q1 := search.NewQueryBuilder().AddExactMatches(search.ClusterID, resolver.data.GetMetadata().GetClusterId()).
		AddExactMatches(search.Namespace, resolver.data.Metadata.GetName()).
		AddStrings(search.ViolationState, storage.ViolationState_ACTIVE.String()).ProtoQuery()
	q2 := search.NewQueryBuilder().AddStrings(search.LifecycleStage, storage.LifecycleStage_DEPLOY.String()).ProtoQuery()
	cq := search.NewConjunctionQuery(q1, q2)
	cq.Pagination = &v1.Pagination{Limit: 1}
	results, err := resolver.root.ViolationsDataStore.Search(ctx, cq)
	if err != nil {
		return false, err
	}
	return len(results) == 0, nil
}
