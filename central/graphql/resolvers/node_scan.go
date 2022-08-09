package resolvers

import (
	"context"

	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddExtraResolvers("NodeScan", []string{
			// NOTE: This list is and should remain alphabetically ordered
			"nodeComponentCount(query: String): Int!",
			"nodeComponents(query: String, pagination: Pagination): [NodeComponent!]!",
		}),
		// deprecated fields
		schema.AddExtraResolvers("NodeScan", []string{
			"componentCount(query: String): Int! " +
				"@deprecated(reason: \"use 'nodeComponentCount'\")",
			"components(query: String, pagination: Pagination): [EmbeddedNodeScanComponent!]! " +
				"@deprecated(reason: \"use 'nodeComponents'\")",
		}),
	)
}

func (resolver *nodeScanResolver) NodeComponentCount(_ context.Context, args RawQuery) (int32, error) {
	return resolver.root.NodeComponentCount(resolver.ctx, args)
}

func (resolver *nodeScanResolver) NodeComponents(_ context.Context, args PaginatedQuery) ([]NodeComponentResolver, error) {
	return resolver.root.NodeComponents(resolver.ctx, args)
}
