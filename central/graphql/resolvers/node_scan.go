package resolvers

import (
	"context"
	"sort"

	"github.com/stackrox/rox/central/graphql/resolvers/embeddedobjs"
	"github.com/stackrox/rox/central/node/datastore/store/common/v2"
	"github.com/stackrox/rox/central/node/mappings"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/scancomponent"
	"github.com/stackrox/rox/pkg/search"
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
	query, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}
	return getNodeComponentResolvers(resolver.ctx, resolver.root, resolver.data, query)
}

func getNodeComponentResolvers(ctx context.Context, root *Resolver, nodeScan *storage.NodeScan, query *v1.Query) ([]NodeComponentResolver, error) {
	query, _ = search.FilterQueryWithMap(query, mappings.NodeComponentOptionsMap)
	predicate, err := nodeComponentPredicateFactory.GeneratePredicate(query)
	if err != nil {
		return nil, err
	}

	idToComponent := make(map[string]*nodeComponentResolver)
	for _, embeddedComponent := range nodeScan.GetComponents() {
		if !predicate.Matches(embeddedComponent) {
			continue
		}

		os := nodeScan.GetOperatingSystem()
		id := scancomponent.ComponentID(embeddedComponent.GetName(), embeddedComponent.GetVersion(), os)
		if _, exists := idToComponent[id]; !exists {
			component := common.GenerateNodeComponent(os, embeddedComponent)
			resolver, err := root.wrapNodeComponent(component, true, nil)
			if err != nil {
				return nil, err
			}
			resolver.ctx = embeddedobjs.NodeComponentContext(ctx, os, nodeScan.GetScanTime(), embeddedComponent)
			idToComponent[id] = resolver
		}
	}

	// For now, sort by IDs.
	resolvers := make([]*nodeComponentResolver, 0, len(idToComponent))
	for _, component := range idToComponent {
		resolvers = append(resolvers, component)
	}
	if len(query.GetPagination().GetSortOptions()) == 0 {
		sort.SliceStable(resolvers, func(i, j int) bool {
			return resolvers[i].data.GetId() < resolvers[j].data.GetId()
		})
	}
	resolverI := make([]NodeComponentResolver, 0, len(resolvers))
	for _, resolver := range resolvers {
		resolverI = append(resolverI, resolver)
	}
	ret, err := paginationWrapper{
		pv: query.GetPagination(),
	}.paginate(resolverI, nil)
	return ret.([]NodeComponentResolver), err
}
