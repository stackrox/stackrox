package resolvers

import (
	"context"

	"github.com/stackrox/rox/central/views/common"
	"github.com/stackrox/rox/central/views/platformcve"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddType("PlatformCVECountByFixability", []string{
			"total: Int!",
			"fixable: Int!",
		}),
	)
}

type platformCVECountByFixabilityResolver struct {
	ctx  context.Context
	root *Resolver
	data common.ResourceCountByFixability
}

func (resolver *Resolver) wrapPlatformCVECountByFixabilityWithContext(ctx context.Context,
	value common.ResourceCountByFixability, err error) (*platformCVECountByFixabilityResolver, error) {
	if err != nil {
		return nil, err
	}
	if value == nil {
		return &platformCVECountByFixabilityResolver{ctx: ctx, root: resolver, data: platformcve.NewEmptyCVECountByFixability()}, nil
	}
	return &platformCVECountByFixabilityResolver{ctx: ctx, root: resolver, data: value}, nil
}

// Total returns the total number of platform CVEs
func (resolver *platformCVECountByFixabilityResolver) Total(_ context.Context) int32 {
	return int32(resolver.data.GetTotal())
}

// Fixable returns the number of fixable platform CVEs
func (resolver *platformCVECountByFixabilityResolver) Fixable(_ context.Context) int32 {
	return int32(resolver.data.GetFixable())
}
