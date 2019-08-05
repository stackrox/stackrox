package resolvers

import (
	"context"

	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/rox/central/namespace"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddQuery("namespaces(query: String): [Namespace!]!"),
		schema.AddQuery("namespace(id: ID!): Namespace"),
		schema.AddQuery("namespaceByClusterIDAndName(clusterID: ID!, name: String!): Namespace"),
		schema.AddExtraResolver("Namespace", "complianceResults(query: String): [ControlResult!]!"),
		schema.AddExtraResolver("Namespace", `subjects(query: String): [Subject!]!`),
		schema.AddExtraResolver("Namespace", `subjectCount: Int!`),
		schema.AddExtraResolver("Namespace", `serviceAccountCount: Int!`),
		schema.AddExtraResolver("Namespace", `k8sroleCount: Int!`),
		schema.AddExtraResolver("Namespace", `policyCount: Int!`),
		schema.AddExtraResolver("Namespace", `policyStatus: PolicyStatus!`),
		schema.AddExtraResolver("Namespace", `policies(query: String): [Policy!]!`),
		schema.AddExtraResolver("Namespace", `images(query: String): [Image!]!`),
		schema.AddExtraResolver("Namespace", `imageCount: Int!`),
		schema.AddExtraResolver("Namespace", `secrets(query: String): [Secret!]!`),
		schema.AddExtraResolver("Namespace", `deployments(query: String): [Deployment!]!`),
		schema.AddExtraResolver("Namespace", "cluster: Cluster!"),
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

// SubjectCount returns the count of Subjects which have any permission on this namespace or the cluster it belongs to
func (resolver *namespaceResolver) SubjectCount(ctx context.Context) (int32, error) {
	if err := readK8sSubjects(ctx); err != nil {
		return 0, err
	}
	if err := readK8sRoleBindings(ctx); err != nil {
		return 0, err
	}
	subjects, err := resolver.getSubjects(ctx, search.EmptyQuery())
	if err != nil {
		return 0, err
	}
	return int32(len(subjects)), nil
}

// Subjects returns the Subjects which have any permission in namespace or cluster wide
func (resolver *namespaceResolver) Subjects(ctx context.Context, args rawQuery) ([]*subjectResolver, error) {
	if err := readK8sSubjects(ctx); err != nil {
		return nil, err
	}
	if err := readK8sRoleBindings(ctx); err != nil {
		return nil, err
	}
	var resolvers []*subjectResolver
	baseQuery, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}
	subjects, err := resolver.getSubjects(ctx, baseQuery)
	if err != nil {
		return nil, err
	}
	for _, subject := range subjects {
		resolvers = append(resolvers, &subjectResolver{resolver.root, subject})
	}
	return resolvers, nil
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

func (resolver *namespaceResolver) filterPoliciesApplicableToNamespace(policies []*storage.Policy) []*storage.Policy {
	var filteredPolicies []*storage.Policy
	clusterID := resolver.data.GetMetadata().GetClusterId()
	for _, policy := range policies {
		if resolver.policyAppliesToNamespace(policy, clusterID) {
			filteredPolicies = append(filteredPolicies, policy)
		}
	}
	return filteredPolicies
}

func (resolver *namespaceResolver) getNamespacePolicies(ctx context.Context) ([]*storage.Policy, error) {
	if err := readPolicies(ctx); err != nil {
		return nil, err
	}
	policies, err := resolver.root.PolicyDataStore.GetPolicies(ctx)
	if err != nil {
		return nil, err
	}
	return resolver.filterPoliciesApplicableToNamespace(policies), nil
}

// PolicyCount returns count of policies applicable to this namespace
func (resolver *namespaceResolver) PolicyCount(ctx context.Context) (int32, error) {
	policies, err := resolver.getNamespacePolicies(ctx)
	if err != nil {
		return 0, err
	}
	return int32(len(policies)), nil
}

// Policies returns all the policies applicable to this namespace
func (resolver *namespaceResolver) Policies(ctx context.Context, args rawQuery) ([]*policyResolver, error) {
	if err := readPolicies(ctx); err != nil {
		return nil, err
	}
	q, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}
	policies, err := resolver.root.PolicyDataStore.SearchRawPolicies(ctx, q)
	if err != nil {
		return nil, err
	}
	return resolver.root.wrapPolicies(resolver.filterPoliciesApplicableToNamespace(policies), nil)
}

func (resolver *namespaceResolver) policyAppliesToNamespace(policy *storage.Policy, clusterID string) bool {
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
func (resolver *namespaceResolver) PolicyStatus(ctx context.Context) (*policyStatusResolver, error) {

	alerts, err := resolver.getActiveDeployAlerts(ctx)
	if err != nil {
		return nil, err
	}

	if len(alerts) == 0 {
		return &policyStatusResolver{"pass", nil}, nil
	}

	policyIDs := set.NewStringSet()
	for _, alert := range alerts {
		policyIDs.Add(alert.GetPolicy().GetId())
	}

	policies, err := resolver.root.wrapPolicies(
		resolver.root.PolicyDataStore.SearchRawPolicies(ctx, search.NewQueryBuilder().AddDocIDs(policyIDs.AsSlice()...).ProtoQuery()))

	if err != nil {
		return nil, err
	}

	return &policyStatusResolver{"fail", policies}, nil
}

func (resolver *namespaceResolver) getActiveDeployAlerts(ctx context.Context) ([]*storage.ListAlert, error) {
	if err := readAlerts(ctx); err != nil {
		return nil, err
	}

	namespace := resolver.data

	q := search.NewQueryBuilder().AddExactMatches(search.ClusterID, namespace.GetMetadata().GetClusterId()).
		AddExactMatches(search.Namespace, namespace.GetMetadata().GetName()).
		AddStrings(search.ViolationState, storage.ViolationState_ACTIVE.String()).
		AddStrings(search.LifecycleStage, storage.LifecycleStage_DEPLOY.String()).ProtoQuery()

	return resolver.root.ViolationsDataStore.SearchListAlerts(ctx, q)
}

func (resolver *namespaceResolver) Images(ctx context.Context, args rawQuery) ([]*imageResolver, error) {
	if err := readImages(ctx); err != nil {
		return nil, err
	}
	q, err := resolver.getClusterNamespaceQuery(args)
	if err != nil {
		return nil, err
	}
	return resolver.root.wrapImages(resolver.root.ImageDataStore.SearchRawImages(ctx, q))
}

func (resolver *namespaceResolver) Secrets(ctx context.Context, args rawQuery) ([]*secretResolver, error) {
	if err := readSecrets(ctx); err != nil {
		return nil, err
	}
	q, err := resolver.getClusterNamespaceQuery(args)
	if err != nil {
		return nil, err
	}
	return resolver.root.wrapSecrets(resolver.root.SecretsDataStore.SearchRawSecrets(ctx, q))
}

func (resolver *namespaceResolver) Deployments(ctx context.Context, args rawQuery) ([]*deploymentResolver, error) {
	if err := readDeployments(ctx); err != nil {
		return nil, err
	}
	q, err := resolver.getClusterNamespaceQuery(args)
	if err != nil {
		return nil, err
	}
	return resolver.root.wrapDeployments(resolver.root.DeploymentDataStore.SearchRawDeployments(ctx, q))
}

func (resolver *namespaceResolver) Cluster(ctx context.Context) (*clusterResolver, error) {
	if err := readClusters(ctx); err != nil {
		return nil, err
	}
	return resolver.root.wrapCluster(resolver.root.ClusterDataStore.GetCluster(ctx, resolver.data.GetMetadata().GetClusterId()))
}

func (resolver *namespaceResolver) getClusterNamespaceQuery(args rawQuery) (*v1.Query, error) {
	q1, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}
	q2 := search.NewQueryBuilder().AddExactMatches(search.ClusterID, resolver.data.GetMetadata().GetClusterId()).
		AddExactMatches(search.Namespace, resolver.data.Metadata.GetName()).ProtoQuery()
	return search.NewConjunctionQuery(q1, q2), nil
}
