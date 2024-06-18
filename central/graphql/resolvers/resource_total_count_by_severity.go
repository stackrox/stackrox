package resolvers

import (
	"context"

	"github.com/stackrox/rox/central/views/common"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddType("ResourceTotalCountByCVESeverity", []string{
			"critical: Int!",
			"important: Int!",
			"moderate: Int!",
			"low: Int!",
		}),
	)
}

// resourceTotalCountBySeverityResolver is an alternative resolver that provides only access to the total count
// of the resource by severity.
type resourceTotalCountBySeverityResolver struct {
	ctx  context.Context
	root *Resolver
	data common.ResourceCountByCVESeverity
}

func (resolver *Resolver) wrapResourceTotalCountBySeverityContext(ctx context.Context, value common.ResourceCountByCVESeverity, err error) (*resourceTotalCountBySeverityResolver, error) {
	if err != nil {
		return nil, err
	}
	if value == nil {
		return &resourceTotalCountBySeverityResolver{ctx: ctx, root: resolver, data: common.NewEmptyResourceCountByCVESeverity()}, nil
	}
	return &resourceTotalCountBySeverityResolver{ctx: ctx, root: resolver, data: value}, nil
}

// Critical returns the total number of resource with low CVE impact.
func (resolver *resourceTotalCountBySeverityResolver) Critical(_ context.Context) int32 {
	value := resolver.data.GetCriticalSeverityCount()
	if value == nil {
		return int32(0)
	}

	return int32(value.GetTotal())
}

// Important returns the total number of resource with important CVE impact.
func (resolver *resourceTotalCountBySeverityResolver) Important(ctx context.Context) int32 {
	value := resolver.data.GetImportantSeverityCount()
	if value == nil {
		return int32(0)
	}

	return int32(value.GetTotal())
}

// Moderate returns the total number of resource with moderate CVE impact.
func (resolver *resourceTotalCountBySeverityResolver) Moderate(ctx context.Context) int32 {
	value := resolver.data.GetModerateSeverityCount()
	if value == nil {
		return int32(0)
	}

	return int32(value.GetTotal())
}

// Low returns the total number of resource with low CVE impact.
func (resolver *resourceTotalCountBySeverityResolver) Low(ctx context.Context) int32 {
	value := resolver.data.GetLowSeverityCount()
	if value == nil {
		return int32(0)
	}

	return int32(value.GetTotal())
}
