package resolvers

import (
	"context"

	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search/scoped"
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
	scope, ok := scoped.GetScope(resolver.ctx)
	if !ok {
		return nil, errors.New("ImageScan.ImageComponents called without scope")
	} else if scope.Level != v1.SearchCategory_IMAGES {
		return nil, errors.New("ImageScan.ImageComponents called with improper scope context")
	}

	return resolver.root.ImageComponents(resolver.ctx, args)
}

func (resolver *imageScanResolver) ImageComponentCount(_ context.Context, args RawQuery) (int32, error) {
	return resolver.root.ImageComponentCount(resolver.ctx, args)
}
