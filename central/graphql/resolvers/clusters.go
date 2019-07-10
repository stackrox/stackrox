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
		schema.AddType("SubjectWithClusterID", []string{"clusterID: String!", "subject: Subject!"}),
		schema.AddQuery("clusters(query: String): [Cluster!]!"),
		schema.AddQuery("cluster(id: ID!): Cluster"),
		schema.AddExtraResolver("Cluster", `alerts: [Alert!]!`),
		schema.AddExtraResolver("Cluster", `alertsCount: Int!`),
		schema.AddExtraResolver("Cluster", `deployments: [Deployment!]!`),
		schema.AddExtraResolver("Cluster", `nodes: [Node!]!`),
		schema.AddExtraResolver("Cluster", `nodeCount: Int!`),
		schema.AddExtraResolver("Cluster", `node(node: ID!): Node`),
		schema.AddExtraResolver("Cluster", `namespaces: [Namespace!]!`),
		schema.AddExtraResolver("Cluster", `namespace(name: String!): Namespace`),
		schema.AddExtraResolver("Cluster", "complianceResults(query: String): [ControlResult!]!"),
		schema.AddExtraResolver("Cluster", `k8sroles: [K8SRole!]!`),
		schema.AddExtraResolver("Cluster", `k8srole(role: ID!): K8SRole`),
		schema.AddExtraResolver("Cluster", `k8sroleCount: Int!`),
		schema.AddExtraResolver("Cluster", `serviceAccounts: [ServiceAccount!]!`),
		schema.AddExtraResolver("Cluster", `serviceAccount(sa: ID!): ServiceAccount`),
		schema.AddExtraResolver("Cluster", `serviceAccountCount: Int!`),
		schema.AddExtraResolver("Cluster", `subjects: [SubjectWithClusterID!]!`),
		schema.AddExtraResolver("Cluster", `subject(name: String!): SubjectWithClusterID!`),
		schema.AddExtraResolver("Cluster", `subjectCount: Int!`),
		schema.AddExtraResolver("Cluster", `images: [Image!]!`),
		schema.AddExtraResolver("Cluster", `imageCount: Int!`),
		schema.AddExtraResolver("Cluster", `policies: [Policy!]!`),
		schema.AddExtraResolver("Cluster", `policyCount: Int!`),
		schema.AddExtraResolver("Cluster", `policyStatus: Boolean!`),
		schema.AddExtraResolver("Cluster", `secrets: [Secret!]!`),
		schema.AddExtraResolver("Cluster", `secretCount: Int!`),
	)
}

// Cluster returns a GraphQL resolver for the given cluster
func (resolver *Resolver) Cluster(ctx context.Context, args struct{ graphql.ID }) (*clusterResolver, error) {
	if err := readClusters(ctx); err != nil {
		return nil, err
	}
	return resolver.wrapCluster(resolver.ClusterDataStore.GetCluster(ctx, string(args.ID)))
}

// Clusters returns GraphQL resolvers for all clusters
func (resolver *Resolver) Clusters(ctx context.Context, args rawQuery) ([]*clusterResolver, error) {
	if err := readClusters(ctx); err != nil {
		return nil, err
	}
	query, err := args.AsV1Query()
	if err != nil {
		return nil, err
	}
	if query == nil {
		return resolver.wrapClusters(resolver.ClusterDataStore.GetClusters(ctx))
	}
	return resolver.wrapClusters(resolver.ClusterDataStore.SearchRawClusters(ctx, query))
}

// Alerts returns GraphQL resolvers for all alerts on this cluster
func (resolver *clusterResolver) Alerts(ctx context.Context) ([]*alertResolver, error) {
	if err := readAlerts(ctx); err != nil {
		return nil, err // could return nil, nil to prevent errors from propagating.
	}
	query := search.NewQueryBuilder().AddStrings(search.ClusterID, resolver.data.GetId()).ProtoQuery()
	return resolver.root.wrapAlerts(
		resolver.root.ViolationsDataStore.SearchRawAlerts(ctx, query))
}

func (resolver *clusterResolver) AlertsCount(ctx context.Context) (int32, error) {
	if err := readAlerts(ctx); err != nil {
		return 0, err
	}
	query := search.NewQueryBuilder().AddStrings(search.ClusterID, resolver.data.GetId()).ProtoQuery()
	results, err := resolver.root.ViolationsDataStore.Search(ctx, query)
	if err != nil {
		return 0, err
	}

	return int32(len(results)), nil
}

// Deployments returns GraphQL resolvers for all deployments in this cluster
func (resolver *clusterResolver) Deployments(ctx context.Context) ([]*deploymentResolver, error) {
	if err := readDeployments(ctx); err != nil {
		return nil, err
	}
	query := search.NewQueryBuilder().AddStrings(search.ClusterID, resolver.data.GetId()).ProtoQuery()
	return resolver.root.wrapDeployments(
		resolver.root.DeploymentDataStore.SearchRawDeployments(ctx, query))
}

// Nodes returns all nodes on the cluster
func (resolver *clusterResolver) Nodes(ctx context.Context) ([]*nodeResolver, error) {
	if err := readNodes(ctx); err != nil {
		return nil, err
	}
	store, err := resolver.root.NodeGlobalDataStore.GetClusterNodeStore(ctx, resolver.data.GetId(), false)
	if err != nil {
		return nil, err
	}
	return resolver.root.wrapNodes(store.ListNodes())
}

// NodeCount returns count of all nodes on the cluster
func (resolver *clusterResolver) NodeCount(ctx context.Context) (int32, error) {
	if err := readNodes(ctx); err != nil {
		return 0, err
	}
	store, err := resolver.root.NodeGlobalDataStore.GetClusterNodeStore(ctx, resolver.data.GetId(), false)
	if err != nil {
		return 0, err
	}

	nodeCount, err := store.CountNodes()
	if err != nil {
		return 0, err
	}

	return int32(nodeCount), nil
}

// Node returns a given node on a cluster
func (resolver *clusterResolver) Node(ctx context.Context, args struct{ Node graphql.ID }) (*nodeResolver, error) {
	if err := readNodes(ctx); err != nil {
		return nil, err
	}
	store, err := resolver.root.NodeGlobalDataStore.GetClusterNodeStore(ctx, resolver.data.GetId(), false)
	if err != nil {
		return nil, err
	}
	node, err := store.GetNode(string(args.Node))
	return resolver.root.wrapNode(node, node != nil, err)
}

// Namespace returns a given namespace on a cluster.
func (resolver *clusterResolver) Namespaces(ctx context.Context) ([]*namespaceResolver, error) {
	if err := readNamespaces(ctx); err != nil {
		return nil, err
	}
	return resolver.root.wrapNamespaces(namespace.ResolveByClusterID(ctx, resolver.data.GetId(),
		resolver.root.NamespaceDataStore, resolver.root.DeploymentDataStore, resolver.root.SecretsDataStore,
		resolver.root.NetworkPoliciesStore))
}

// Namespace returns a given namespace on a cluster.
func (resolver *clusterResolver) Namespace(ctx context.Context, args struct{ Name string }) (*namespaceResolver, error) {
	return resolver.root.NamespaceByClusterIDAndName(ctx, clusterIDAndNameQuery{
		ClusterID: graphql.ID(resolver.data.GetId()),
		Name:      args.Name,
	})
}

func (resolver *clusterResolver) ComplianceResults(ctx context.Context, args rawQuery) ([]*controlResultResolver, error) {
	if err := readCompliance(ctx); err != nil {
		return nil, err
	}

	runResults, err := resolver.root.ComplianceAggregator.GetResultsWithEvidence(ctx, args.String())
	if err != nil {
		return nil, err
	}
	output := newBulkControlResults()
	output.addClusterData(resolver.root, runResults, nil)
	output.addDeploymentData(resolver.root, runResults, nil)
	output.addNodeData(resolver.root, runResults, nil)
	return *output, nil
}

// K8sRoles returns GraphQL resolvers for all k8s roles
func (resolver *clusterResolver) K8sRoles(ctx context.Context) ([]*k8SRoleResolver, error) {
	if err := readK8sRoles(ctx); err != nil {
		return nil, err
	}

	q := search.NewQueryBuilder().AddExactMatches(search.ClusterID, resolver.data.GetId()).ProtoQuery()
	return resolver.root.wrapK8SRoles(resolver.root.K8sRoleStore.SearchRawRoles(ctx, q))
}

// K8sRoleCount returns count of K8s roles in this cluster
func (resolver *clusterResolver) K8sRoleCount(ctx context.Context) (int32, error) {
	if err := readK8sRoles(ctx); err != nil {
		return 0, err
	}
	q := resolver.getClusterQuery()
	results, err := resolver.root.K8sRoleStore.Search(ctx, q)
	if err != nil {
		return 0, err
	}
	return int32(len(results)), nil
}

// K8sRole returns clusterResolver GraphQL resolver for a given k8s role
func (resolver *clusterResolver) K8sRole(ctx context.Context, args struct{ Role graphql.ID }) (*k8SRoleResolver, error) {
	if err := readK8sRoles(ctx); err != nil {
		return nil, err
	}

	q := search.NewQueryBuilder().AddExactMatches(search.ClusterID, resolver.data.GetId()).
		AddExactMatches(search.RoleID, string(args.Role)).ProtoQuery()

	roles, err := resolver.root.K8sRoleStore.SearchRawRoles(ctx, q)

	if err != nil {
		return nil, err
	}

	if len(roles) == 0 {
		return nil, nil
	}

	return resolver.root.wrapK8SRole(roles[0], true, nil)
}

// ServiceAccounts returns GraphQL resolvers for all service accounts in this cluster
func (resolver *clusterResolver) ServiceAccounts(ctx context.Context) ([]*serviceAccountResolver, error) {
	if err := readServiceAccounts(ctx); err != nil {
		return nil, err
	}

	q := search.NewQueryBuilder().AddExactMatches(search.ClusterID, resolver.data.GetId()).ProtoQuery()
	return resolver.root.wrapServiceAccounts(resolver.root.ServiceAccountsDataStore.SearchRawServiceAccounts(ctx, q))
}

// ServiceAccountCount returns count of Service Accounts in this cluster
func (resolver *clusterResolver) ServiceAccountCount(ctx context.Context) (int32, error) {
	if err := readServiceAccounts(ctx); err != nil {
		return 0, err
	}
	q := resolver.getClusterQuery()
	results, err := resolver.root.ServiceAccountsDataStore.Search(ctx, q)
	if err != nil {
		return 0, err
	}
	return int32(len(results)), nil
}

// ServiceAccount returns clusterResolver GraphQL resolver for a given service account
func (resolver *clusterResolver) ServiceAccount(ctx context.Context, args struct{ Sa graphql.ID }) (*serviceAccountResolver, error) {
	if err := readK8sRoles(ctx); err != nil {
		return nil, err
	}

	q := search.NewQueryBuilder().AddExactMatches(search.ClusterID, resolver.data.GetId()).
		AddExactMatches(search.RoleID, string(args.Sa)).ProtoQuery()

	serviceAccounts, err := resolver.root.ServiceAccountsDataStore.SearchRawServiceAccounts(ctx, q)

	if err != nil {
		return nil, err
	}

	if len(serviceAccounts) == 0 {
		return nil, nil
	}

	return resolver.root.wrapServiceAccount(serviceAccounts[0], true, nil)
}

// Subjects returns GraphQL resolvers for all subjects in this cluster
func (resolver *clusterResolver) Subjects(ctx context.Context) ([]*subjectWithClusterIDResolver, error) {
	subjectResolvers, err := resolver.root.wrapSubjects(resolver.getClusterSubjects(ctx))
	if err != nil {
		return nil, err
	}
	return wrapSubjects(resolver.data.GetId(), subjectResolvers), nil
}

// SubjectCount returns count of Users and Groups in this cluster
func (resolver *clusterResolver) SubjectCount(ctx context.Context) (int32, error) {
	subjects, err := resolver.getClusterSubjects(ctx)
	if err != nil {
		return 0, err
	}
	return int32(len(subjects)), nil
}

// ServiceAccount returns clusterResolver GraphQL resolver for a given service account
func (resolver *clusterResolver) Subject(ctx context.Context, args struct{ Name string }) (*subjectWithClusterIDResolver, error) {
	bindings, err := resolver.getRoleBindings(ctx)
	if err != nil {
		return nil, err
	}
	subject, err := resolver.root.wrapSubject(k8srbac.GetSubject(args.Name, bindings))
	if err != nil {
		return nil, err
	}
	return wrapSubject(resolver.data.GetId(), subject), nil
}

func (resolver *clusterResolver) Images(ctx context.Context) ([]*imageResolver, error) {
	if err := readImages(ctx); err != nil {
		return nil, err
	}
	q := search.NewQueryBuilder().AddExactMatches(search.ClusterID, resolver.data.GetId()).ProtoQuery()
	return resolver.root.wrapListImages(resolver.root.ImageDataStore.SearchListImages(ctx, q))
}

func (resolver *clusterResolver) ImageCount(ctx context.Context) (int32, error) {
	if err := readImages(ctx); err != nil {
		return 0, err
	}
	q := search.NewQueryBuilder().AddExactMatches(search.ClusterID, resolver.data.GetId()).ProtoQuery()
	results, err := resolver.root.ImageDataStore.Search(ctx, q)
	if err != nil {
		return 0, err
	}
	return int32(len(results)), nil
}

func (resolver *clusterResolver) Policies(ctx context.Context) ([]*policyResolver, error) {
	if err := readPolicies(ctx); err != nil {
		return nil, err
	}

	policies, err := resolver.root.PolicyDataStore.GetPolicies(ctx)
	if err != nil {
		return nil, err
	}
	var filteredPolicies []*storage.Policy
	for _, policy := range policies {
		if resolver.policyAppliesToCluster(ctx, policy) {
			filteredPolicies = append(filteredPolicies, policy)
		}
	}
	return resolver.root.wrapPolicies(filteredPolicies, nil)
}

func (resolver *clusterResolver) policyAppliesToCluster(ctx context.Context, policy *storage.Policy) bool {
	// Global Policy
	if len(policy.Scope) == 0 {
		return true
	}
	// Clustered or namespaced scope policy
	for _, scope := range policy.Scope {
		if scope.GetCluster() != "" {
			if scope.GetCluster() == resolver.data.GetId() {
				return true
			}
		} else if scope.GetNamespace() != "" {
			q := search.NewQueryBuilder().AddExactMatches(search.ClusterID, resolver.data.GetId()).
				AddExactMatches(search.Namespace, scope.GetNamespace()).ProtoQuery()
			result, err := resolver.root.NamespaceDataStore.Search(ctx, q)
			if err != nil {
				continue
			}
			if len(result) != 0 {
				return true
			}
		} else {
			return true
		}
	}
	return false
}

func (resolver *clusterResolver) PolicyCount(ctx context.Context) (int32, error) {
	resolvers, err := resolver.Policies(ctx)
	if err != nil {
		return 0, err
	}
	return int32(len(resolvers)), nil
}

// PolicyStatus returns true if there is no policy violation for this cluster
func (resolver *clusterResolver) PolicyStatus(ctx context.Context) (bool, error) {
	if err := readAlerts(ctx); err != nil {
		return false, err // could return nil, nil to prevent errors from propagating.
	}
	q1 := search.NewQueryBuilder().AddExactMatches(search.ClusterID, resolver.data.GetId()).
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

func (resolver *clusterResolver) Secrets(ctx context.Context) ([]*secretResolver, error) {
	query := resolver.getClusterQuery()
	return resolver.root.wrapListSecrets(resolver.root.SecretsDataStore.SearchListSecrets(ctx, query))
}

func (resolver *clusterResolver) SecretCount(ctx context.Context) (int32, error) {
	query := resolver.getClusterQuery()
	result, err := resolver.root.SecretsDataStore.Search(ctx, query)
	if err != nil {
		return 0, err
	}
	return int32(len(result)), nil
}
