package resolvers

import (
	"context"

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
	)
}

func (resolver *imageScanResolver) ImageComponents(_ context.Context, args PaginatedQuery) ([]ImageComponentResolver, error) {
	return resolver.root.ImageComponents(resolver.ctx, args)
}

func (resolver *imageScanResolver) ImageComponentCount(_ context.Context, args RawQuery) (int32, error) {
	return resolver.root.ImageComponentCount(resolver.ctx, args)
}
