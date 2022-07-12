package resolvers

import (
	"context"

	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddType("PlottedImageVulnerabilities", []string{
			"basicImageVulnerabilityCounter: VulnerabilityCounter!",
			"imageVulnerabilities(pagination: Pagination): [ImageVulnerability]!",
		}),
	)
}

// PlottedImageVulnerabilitiesResolver returns the data required by top risky images scatter-plot on vuln mgmt dashboard
type PlottedImageVulnerabilitiesResolver struct {
	root    *Resolver
	all     []string
	fixable int
}

func newPlottedImageVulnerabilitiesResolver(ctx context.Context, root *Resolver, args RawQuery) (*PlottedImageVulnerabilitiesResolver, error) {
	query := withImageCveTypeFiltering(args.String())
	allCveIds, fixableCount, err := getPlottedVulnsIdsAndFixableCount(ctx, root, RawQuery{Query: &query})
	if err != nil {
		return nil, err
	}

	return &PlottedImageVulnerabilitiesResolver{
		root:    root,
		all:     allCveIds,
		fixable: fixableCount,
	}, nil
}

// BasicImageVulnerabilityCounter returns the ImageVulnerabilityCounter for scatter-plot with only total and fixable
func (pvr *PlottedImageVulnerabilitiesResolver) BasicImageVulnerabilityCounter(_ context.Context) (*VulnerabilityCounterResolver, error) {
	return &VulnerabilityCounterResolver{
		all: &VulnerabilityFixableCounterResolver{
			total:   int32(len(pvr.all)),
			fixable: int32(pvr.fixable),
		},
	}, nil
}

// ImageVulnerabilities returns the image vulnerabilities for top risky images scatter-plot
func (pvr *PlottedImageVulnerabilitiesResolver) ImageVulnerabilities(ctx context.Context, args PaginatedQuery) ([]ImageVulnerabilityResolver, error) {
	vulnResolvers, err := unwrappedPlottedVulnerabilities(ctx, pvr.root, pvr.all, args)
	if err != nil {
		return nil, err
	}

	ret := make([]ImageVulnerabilityResolver, 0, len(vulnResolvers))
	for _, resolver := range vulnResolvers {
		ret = append(ret, resolver)
	}
	return ret, nil
}
