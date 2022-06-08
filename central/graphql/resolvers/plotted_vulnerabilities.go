package resolvers

import (
	"context"

	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddType("PlottedVulnerabilities", []string{
			"basicVulnCounter: VulnerabilityCounter!",
			"vulns(pagination: Pagination): [EmbeddedVulnerability]!",
		}),
	)
}

// PlottedVulnerabilitiesResolver returns the data required by top risky entity scatter-plot on vuln mgmt dashboard
type PlottedVulnerabilitiesResolver struct {
	root    *Resolver
	all     []string
	fixable int
}

func newPlottedVulnerabilitiesResolver(ctx context.Context, root *Resolver, args RawQuery) (*PlottedVulnerabilitiesResolver, error) {
	allCveIds, fixableCount, err := getPlottedVulnsIdsAndFixableCount(ctx, root, args)
	if err != nil {
		return nil, err
	}

	return &PlottedVulnerabilitiesResolver{
		root:    root,
		all:     allCveIds,
		fixable: fixableCount,
	}, nil
}

// BasicVulnCounter returns the vulnCounter for scatter-plot with only total and fixable
func (pvr *PlottedVulnerabilitiesResolver) BasicVulnCounter(ctx context.Context) (*VulnerabilityCounterResolver, error) {
	return &VulnerabilityCounterResolver{
		all: &VulnerabilityFixableCounterResolver{
			total:   int32(len(pvr.all)),
			fixable: int32(pvr.fixable),
		},
	}, nil
}

// Vulns returns the vulns for scatter-plot
func (pvr *PlottedVulnerabilitiesResolver) Vulns(ctx context.Context, args PaginatedQuery) ([]VulnerabilityResolver, error) {
	vulnResolvers, err := unwrappedPlottedVulnerabilities(ctx, pvr.root, pvr.all, args)
	if err != nil {
		return nil, err
	}

	ret := make([]VulnerabilityResolver, 0, len(vulnResolvers))
	for _, resolver := range vulnResolvers {
		ret = append(ret, resolver)
	}
	return ret, nil
}
