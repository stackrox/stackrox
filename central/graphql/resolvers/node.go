package resolvers

import (
	"context"

	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/rox/pkg/utils"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddQuery("node(id:ID!): Node"),
	)
}

// Node returns a resolver for a matching node, or nil if no node is found in any cluster
func (resolver *Resolver) Node(ctx context.Context, args struct{ graphql.ID }) (*nodeResolver, error) {
	if err := readNodes(ctx); err != nil {
		return nil, err
	}
	clusters, err := resolver.ClusterDataStore.GetClusters()
	if err != nil {
		return nil, err
	}
	var output *nodeResolver
	for _, cluster := range clusters {
		store, err := resolver.NodeGlobalStore.GetClusterNodeStore(cluster.GetId())
		if err != nil {
			return nil, err
		}
		node, err := store.GetNode(string(args.ID))
		if err != nil {
			return nil, err
		}
		if node != nil {
			if output == nil {
				output = &nodeResolver{root: resolver, data: node}
			} else {
				return nil, status.Error(codes.Internal, "multiple matching node ids found")
			}
		}
	}
	return output, nil
}
