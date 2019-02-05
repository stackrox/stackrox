package resolvers

import (
	"context"

	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/rox/central/namespace"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddQuery("clusters: [Cluster!]!"),
		schema.AddQuery("cluster(id: ID!): Cluster"),

		schema.AddExtraResolver("Cluster", `alerts: [Alert!]!`),
		schema.AddExtraResolver("Cluster", `deployments: [Deployment!]!`),
		schema.AddExtraResolver("Cluster", `nodes: [Node!]!`),
		schema.AddExtraResolver("Cluster", `node(node: ID!): Node`),
		schema.AddExtraResolver("Cluster", `namespaces: [Namespace!]!`),
		schema.AddExtraResolver("Cluster", `namespace(name: String!): Namespace`),
	)
}

// Cluster returns a GraphQL resolver for the given cluster
func (resolver *Resolver) Cluster(ctx context.Context, args struct{ graphql.ID }) (*clusterResolver, error) {
	if err := readClusters(ctx); err != nil {
		return nil, err
	}
	return resolver.wrapCluster(resolver.ClusterDataStore.GetCluster(string(args.ID)))
}

// Clusters returns GraphQL resolvers for all clusters
func (resolver *Resolver) Clusters(ctx context.Context) ([]*clusterResolver, error) {
	if err := readClusters(ctx); err != nil {
		return nil, err
	}
	return resolver.wrapClusters(resolver.ClusterDataStore.GetClusters())
}

// Alerts returns GraphQL resolvers for all alerts on this cluster
func (resolver *clusterResolver) Alerts(ctx context.Context) ([]*alertResolver, error) {
	if err := readAlerts(ctx); err != nil {
		return nil, err // could return nil, nil to prevent errors from propagating.
	}
	query := search.NewQueryBuilder().AddStrings(search.ClusterID, resolver.data.GetId()).ProtoQuery()
	return resolver.root.wrapAlerts(
		resolver.root.ViolationsDataStore.SearchRawAlerts(query))
}

// Deployments returns GraphQL resolvers for all deployments in this cluster
func (resolver *clusterResolver) Deployments(ctx context.Context) ([]*deploymentResolver, error) {
	if err := readDeployments(ctx); err != nil {
		return nil, err
	}
	query := search.NewQueryBuilder().AddStrings(search.ClusterID, resolver.data.GetId()).ProtoQuery()
	return resolver.root.wrapDeployments(
		resolver.root.DeploymentDataStore.SearchRawDeployments(query))
}

// Nodes returns all nodes on the cluster
func (resolver *clusterResolver) Nodes(ctx context.Context) ([]*nodeResolver, error) {
	if err := readNodes(ctx); err != nil {
		return nil, err
	}
	store, err := resolver.root.NodeGlobalStore.GetClusterNodeStore(resolver.data.GetId())
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
	store, err := resolver.root.NodeGlobalStore.GetClusterNodeStore(resolver.data.GetId())
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
	return resolver.root.wrapNamespaces(namespace.ResolveByClusterID(resolver.data.GetId(),
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
