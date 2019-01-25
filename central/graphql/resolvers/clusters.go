package resolvers

import (
	"context"

	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/rox/pkg/search"
)

func init() {
	schema := getBuilder()
	schema.AddQuery("clusters: [Cluster!]!")
	schema.AddQuery("cluster(id: ID!): Cluster")

	schema.AddExtraResolver("Cluster", `alerts: [Alert!]!`)
	schema.AddExtraResolver("Cluster", `deployments: [Deployment!]!`)
	schema.AddExtraResolver("Cluster", `nodes: [Node!]!`)
	schema.AddExtraResolver("Cluster", `node(node: ID!): Node`)
}

// Cluster returns a GraphQL resolver for the given cluster
func (resolver *Resolver) Cluster(ctx context.Context, args struct{ graphql.ID }) (*clusterResolver, error) {
	if err := clusterAuth(ctx); err != nil {
		return nil, err
	}
	return resolver.wrapCluster(resolver.ClusterDataStore.GetCluster(string(args.ID)))
}

// Clusters returns GraphQL resolvers for all clusters
func (resolver *Resolver) Clusters(ctx context.Context) ([]*clusterResolver, error) {
	if err := clusterAuth(ctx); err != nil {
		return nil, err
	}
	return resolver.wrapClusters(resolver.ClusterDataStore.GetClusters())
}

// Alerts returns GraphQL resolvers for all alerts on this cluster
func (resolver *clusterResolver) Alerts(ctx context.Context) ([]*alertResolver, error) {
	if err := alertAuth(ctx); err != nil {
		return nil, err // could return nil, nil to prevent errors from propagating.
	}
	query := search.NewQueryBuilder().AddStrings(search.ClusterID, resolver.data.GetId()).ProtoQuery()
	return resolver.root.wrapAlerts(
		resolver.root.ViolationsDataStore.SearchRawAlerts(query))
}

// Deployments returns GraphQL resolvers for all deployments in this cluster
func (resolver *clusterResolver) Deployments(ctx context.Context) ([]*deploymentResolver, error) {
	if err := deploymentAuth(ctx); err != nil {
		return nil, err
	}
	query := search.NewQueryBuilder().AddStrings(search.ClusterID, resolver.data.GetId()).ProtoQuery()
	return resolver.root.wrapDeployments(
		resolver.root.DeploymentDataStore.SearchRawDeployments(query))
}

// Nodes returns all nodes on the cluster
func (resolver *clusterResolver) Nodes(ctx context.Context) ([]*nodeResolver, error) {
	if err := nodeAuth(ctx); err != nil {
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
	if err := nodeAuth(ctx); err != nil {
		return nil, err
	}
	store, err := resolver.root.NodeGlobalStore.GetClusterNodeStore(resolver.data.GetId())
	if err != nil {
		return nil, err
	}
	node, err := store.GetNode(string(args.Node))
	return resolver.root.wrapNode(node, node != nil, err)
}
