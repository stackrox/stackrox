package resolvers

import (
	"context"

	"github.com/stackrox/rox/central/views/common"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddType("ResourceCountByCVESeverity", []string{
			"critical: ResourceCountByFixability!",
			"important: ResourceCountByFixability!",
			"moderate: ResourceCountByFixability!",
			"low: ResourceCountByFixability!",
		}),
		schema.AddType("ResourceCountByFixability", []string{
			"total: Int!",
			"fixable: Int!",
		}),
	)
}

type resourceCountBySeverityResolver struct {
	ctx  context.Context
	root *Resolver
	data common.ResourceCountByCVESeverity
}

func (resolver *Resolver) wrapResourceCountByCVESeverityWithContext(ctx context.Context, value common.ResourceCountByCVESeverity, err error) (*resourceCountBySeverityResolver, error) {
	if err != nil {
		return nil, err
	}
	if value == nil {
		return &resourceCountBySeverityResolver{ctx: ctx, root: resolver, data: common.NewEmptyResourceCountByCVESeverity()}, nil
	}
	return &resourceCountBySeverityResolver{ctx: ctx, root: resolver, data: value}, nil
}

// Critical returns the number of resource with low CVE impact.
func (resolver *resourceCountBySeverityResolver) Critical(ctx context.Context) (*resourceCountByFixabilityResolver, error) {
	return resolver.root.wrapResourceCountByFixabilityContext(ctx, resolver.data.GetCriticalSeverityCount(), nil)
}

// Important returns the number of resource with important CVE impact.
func (resolver *resourceCountBySeverityResolver) Important(ctx context.Context) (*resourceCountByFixabilityResolver, error) {
	return resolver.root.wrapResourceCountByFixabilityContext(ctx, resolver.data.GetImportantSeverityCount(), nil)
}

// Moderate returns the number of resource with moderate CVE impact.
func (resolver *resourceCountBySeverityResolver) Moderate(ctx context.Context) (*resourceCountByFixabilityResolver, error) {
	return resolver.root.wrapResourceCountByFixabilityContext(ctx, resolver.data.GetModerateSeverityCount(), nil)
}

// Low returns the number of resource with low CVE impact.
func (resolver *resourceCountBySeverityResolver) Low(ctx context.Context) (*resourceCountByFixabilityResolver, error) {
	return resolver.root.wrapResourceCountByFixabilityContext(ctx, resolver.data.GetLowSeverityCount(), nil)
}

type resourceCountByFixabilityResolver struct {
	ctx  context.Context
	root *Resolver
	data common.ResourceCountByFixability
}

func (resolver *Resolver) wrapResourceCountByFixabilityContext(ctx context.Context, value common.ResourceCountByFixability, err error) (*resourceCountByFixabilityResolver, error) {
	if err != nil {
		return nil, err
	}
	if value == nil {
		return &resourceCountByFixabilityResolver{ctx: ctx, root: resolver, data: common.NewEmptyResourceCountByFixability()}, nil
	}
	return &resourceCountByFixabilityResolver{ctx: ctx, root: resolver, data: value}, nil
}

// Total returns the total resource count affected by CVEs.
func (resolver *resourceCountByFixabilityResolver) Total(_ context.Context) int32 {
	return int32(resolver.data.GetTotal())
}

// Fixable returns the number of resource affected by CVEs that are fixable.
func (resolver *resourceCountByFixabilityResolver) Fixable(_ context.Context) int32 {
	return int32(resolver.data.GetFixable())
}
