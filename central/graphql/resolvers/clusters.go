package resolvers

import (
	"context"

	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/rox/central/compliance/store"
	"github.com/stackrox/rox/central/namespace"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/k8srbac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddType("SubjectWithClusterID", []string{"clusterID: String!", "subject: Subject!"}),
		schema.AddQuery("clusters: [Cluster!]!"),
		schema.AddQuery("cluster(id: ID!): Cluster"),

		schema.AddExtraResolver("Cluster", `alerts: [Alert!]!`),
		schema.AddExtraResolver("Cluster", `deployments: [Deployment!]!`),
		schema.AddExtraResolver("Cluster", `nodes: [Node!]!`),
		schema.AddExtraResolver("Cluster", `node(node: ID!): Node`),
		schema.AddExtraResolver("Cluster", `namespaces: [Namespace!]!`),
		schema.AddExtraResolver("Cluster", `namespace(name: String!): Namespace`),
		schema.AddExtraResolver("Cluster", "complianceResults: [ControlResult!]!"),
		schema.AddExtraResolver("Cluster", `k8sroles: [K8SRole!]!`),
		schema.AddExtraResolver("Cluster", `k8srole(role: ID!): K8SRole`),
		schema.AddExtraResolver("Cluster", `serviceAccounts: [ServiceAccount!]!`),
		schema.AddExtraResolver("Cluster", `serviceAccount(sa: ID!): ServiceAccount`),
		schema.AddExtraResolver("Cluster", `subjects: [SubjectWithClusterID!]!`),
		schema.AddExtraResolver("Cluster", `subject(name: String!): SubjectWithClusterID!`),
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
func (resolver *Resolver) Clusters(ctx context.Context) ([]*clusterResolver, error) {
	if err := readClusters(ctx); err != nil {
		return nil, err
	}
	return resolver.wrapClusters(resolver.ClusterDataStore.GetClusters(ctx))
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

func (resolver *clusterResolver) ComplianceResults(ctx context.Context) ([]*controlResultResolver, error) {
	if err := readCompliance(ctx); err != nil {
		return nil, err
	}
	data, err := resolver.root.ComplianceDataStore.GetLatestRunResultsBatch([]string{resolver.data.GetId()}, allStandards(resolver.root.ComplianceStandardStore), store.RequireMessageStrings)
	if err != nil {
		return nil, err
	}
	output := newBulkControlResults()
	output.addClusterData(resolver.root, data, nil)
	output.addDeploymentData(resolver.root, data, nil)
	output.addNodeData(resolver.root, data, nil)
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
	if err := readK8sSubjects(ctx); err != nil {
		return nil, err
	}

	if err := readK8sRoleBindings(ctx); err != nil {
		return nil, err
	}

	q := search.NewQueryBuilder().AddExactMatches(search.ClusterID, resolver.data.GetId()).ProtoQuery()
	bindings, err := resolver.root.K8sRoleBindingStore.SearchRawRoleBindings(ctx, q)

	if err != nil {
		return nil, err
	}

	if len(bindings) == 0 {
		return nil, nil
	}

	subjectResolvers, err := resolver.root.wrapSubjects(k8srbac.GetAllSubjects(bindings,
		storage.SubjectKind_USER, storage.SubjectKind_GROUP), nil)

	if err != nil {
		return nil, err
	}

	return wrapSubjects(resolver.data.GetId(), subjectResolvers), nil
}

// ServiceAccount returns clusterResolver GraphQL resolver for a given service account
func (resolver *clusterResolver) Subject(ctx context.Context, args struct{ Name string }) (*subjectWithClusterIDResolver, error) {
	if err := readK8sRoles(ctx); err != nil {
		return nil, err
	}

	q := search.NewQueryBuilder().AddExactMatches(search.ClusterID, resolver.data.GetId()).ProtoQuery()
	bindings, err := resolver.root.K8sRoleBindingStore.SearchRawRoleBindings(ctx, q)

	if err != nil {
		return nil, err
	}

	if len(bindings) == 0 {
		return nil, nil
	}

	subject, err := resolver.root.wrapSubject(k8srbac.GetSubject(args.Name, bindings))
	if err != nil {
		return nil, err
	}

	return wrapSubject(resolver.data.GetId(), subject), nil
}
