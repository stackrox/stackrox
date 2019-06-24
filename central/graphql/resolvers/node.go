package resolvers

import (
	"context"

	"github.com/graph-gophers/graphql-go"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/utils"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddQuery("node(id:ID!): Node"),
		schema.AddQuery("nodes(query: String): [Node!]!"),
		schema.AddExtraResolver("Node", "complianceResults(query: String): [ControlResult!]!"),
	)
}

// Node returns a resolver for a matching node, or nil if no node is found in any cluster
func (resolver *Resolver) Node(ctx context.Context, args struct{ graphql.ID }) (*nodeResolver, error) {
	if err := readNodes(ctx); err != nil {
		return nil, err
	}
	clusters, err := resolver.ClusterDataStore.GetClusters(ctx)
	if err != nil {
		return nil, err
	}
	var output *nodeResolver
	for _, cluster := range clusters {
		store, err := resolver.NodeGlobalDataStore.GetClusterNodeStore(ctx, cluster.GetId(), false)
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

// Nodes returns resolvers for a matching nodes, or nil if no node is found in any cluster
func (resolver *Resolver) Nodes(ctx context.Context, args rawQuery) ([]*nodeResolver, error) {
	if err := readNodes(ctx); err != nil {
		return nil, err
	}
	query, err := args.AsV1Query()
	if err != nil {
		return nil, err
	}

	var nodeResolvers []*nodeResolver
	nodes, err := resolver.NodeGlobalDataStore.SearchRawNodes(ctx, query)
	if err != nil {
		return nil, err
	}

	for _, node := range nodes {
		nodeResolvers = append(nodeResolvers, &nodeResolver{root: resolver, data: node})
	}

	return nodeResolvers, nil
}

func (resolver *nodeResolver) ComplianceResults(ctx context.Context, args rawQuery) ([]*controlResultResolver, error) {
	if err := readCompliance(ctx); err != nil {
		return nil, err
	}

	runResults, err := resolver.root.ComplianceAggregator.GetResultsWithEvidence(ctx, args.String())
	if err != nil {
		return nil, err
	}
	output := newBulkControlResults()
	nodeID := resolver.data.GetId()
	output.addNodeData(resolver.root, runResults, func(node *storage.Node, _ *v1.ComplianceControl) bool {
		return node.GetId() == nodeID
	})
	return *output, nil
}
