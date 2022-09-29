package resolvers

import (
	"context"
	"sort"

	"github.com/stackrox/rox/central/graphql/resolvers/embeddedobjs"
	"github.com/stackrox/rox/central/image/datastore/store/common/v2"
	"github.com/stackrox/rox/central/image/mappings"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/scancomponent"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddExtraResolvers("ImageScan", []string{
			// NOTE: This list is and should remain alphabetically ordered
			"imageComponentCount(query: String): Int!",
			"imageComponents(query: String, pagination: Pagination): [ImageComponent!]!",
		}),
		// deprecated fields
		schema.AddExtraResolvers("ImageScan", []string{
			"componentCount(query: String): Int! " +
				"@deprecated(reason: \"use 'imageComponentCount'\")",
			"components(query: String, pagination: Pagination): [EmbeddedImageScanComponent!]! " +
				"@deprecated(reason: \"use 'imageComponents'\")",
		}),
	)
}

func (resolver *imageScanResolver) ImageComponents(_ context.Context, args PaginatedQuery) ([]ImageComponentResolver, error) {
	query, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}
	return getImageComponentResolvers(resolver.ctx, resolver.root, resolver.data, query)
}

func (resolver *imageScanResolver) ImageComponentCount(_ context.Context, args RawQuery) (int32, error) {
	return resolver.root.ImageComponentCount(resolver.ctx, args)
}

func getImageComponentResolvers(ctx context.Context, root *Resolver, imageScan *storage.ImageScan, query *v1.Query) ([]ImageComponentResolver, error) {
	query, _ = search.FilterQueryWithMap(query, mappings.ComponentOptionsMap)
	predicate, err := componentPredicateFactory.GeneratePredicate(query)
	if err != nil {
		return nil, err
	}

	idToComponent := make(map[string]*imageComponentResolver)
	for _, embeddedComponent := range imageScan.GetComponents() {
		if !predicate.Matches(embeddedComponent) {
			continue
		}

		os := imageScan.GetOperatingSystem()
		id := scancomponent.ComponentID(embeddedComponent.GetName(), embeddedComponent.GetVersion(), os)
		if _, exists := idToComponent[id]; !exists {
			component := common.GenerateImageComponent(os, embeddedComponent)
			resolver, err := root.wrapImageComponent(component, true, nil)
			if err != nil {
				return nil, err
			}
			resolver.ctx = embeddedobjs.ComponentContext(ctx, os, imageScan.GetScanTime(), embeddedComponent)
			idToComponent[id] = resolver
		}
	}

	// For now, sort by IDs.
	resolvers := make([]*imageComponentResolver, 0, len(idToComponent))
	for _, component := range idToComponent {
		resolvers = append(resolvers, component)
	}
	if len(query.GetPagination().GetSortOptions()) == 0 {
		sort.SliceStable(resolvers, func(i, j int) bool {
			return resolvers[i].data.GetId() < resolvers[j].data.GetId()
		})
	}
	resolverI := make([]ImageComponentResolver, 0, len(resolvers))
	for _, resolver := range resolvers {
		resolverI = append(resolverI, resolver)
	}
	return paginate(query.GetPagination(), resolverI, nil)
}
