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
			"critical: Int!",
			"important: Int!",
			"moderate: Int!",
			"low: Int!",
		}),
	)
}

type resourceCountBySeverityResolver struct {
	ctx  context.Context
	root *Resolver
	data common.ResourceCountByCVESeverity
}

func (resolver *Resolver) wrapResourceCountByCVESeverityWithContext(ctx context.Context, value common.ResourceCountByCVESeverity, err error) (*resourceCountBySeverityResolver, error) {
	if err != nil || value == nil {
		return nil, err
	}
	return &resourceCountBySeverityResolver{ctx: ctx, root: resolver, data: value}, nil
}

// Critical returns the number of resource with low CVE impact.
func (resolver *resourceCountBySeverityResolver) Critical(_ context.Context) int32 {
	return int32(resolver.data.GetCriticalSeverityCount())
}

// Important returns the number of resource with important CVE impact.
func (resolver *resourceCountBySeverityResolver) Important(_ context.Context) int32 {
	return int32(resolver.data.GetImportantSeverityCount())
}

// Moderate returns the number of resource with moderate CVE impact.
func (resolver *resourceCountBySeverityResolver) Moderate(_ context.Context) int32 {
	return int32(resolver.data.GetModerateSeverityCount())
}

// Low returns the number of resource with low CVE impact.
func (resolver *resourceCountBySeverityResolver) Low(_ context.Context) int32 {
	return int32(resolver.data.GetLowSeverityCount())
}
