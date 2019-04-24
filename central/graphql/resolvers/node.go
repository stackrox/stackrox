package resolvers

import (
	"context"

	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/rox/central/compliance/store"
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
		schema.AddExtraResolver("Node", "complianceResults: [ControlResult!]!"),
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

func (resolver *nodeResolver) ComplianceResults(ctx context.Context) ([]*controlResultResolver, error) {
	if err := readCompliance(ctx); err != nil {
		return nil, err
	}
	data, err := resolver.root.ComplianceDataStore.GetLatestRunResultsBatch([]string{resolver.data.GetClusterId()}, allStandards(resolver.root.ComplianceStandardStore), store.RequireMessageStrings)
	if err != nil {
		return nil, err
	}
	output := newBulkControlResults()
	nodeID := resolver.data.GetId()
	output.addNodeData(resolver.root, data, func(node *storage.Node, _ *v1.ComplianceControl) bool {
		return node.GetId() == nodeID
	})
	return *output, nil
}
