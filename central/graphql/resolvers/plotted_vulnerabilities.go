package resolvers

import (
	"context"

	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/pkg/search"
	"github.com/stackrox/stackrox/pkg/utils"
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
	q, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}

	q = tryUnsuppressedQuery(q)
	q.Pagination = &v1.QueryPagination{
		SortOptions: []*v1.QuerySortOption{
			{
				Field:    search.CVSS.String(),
				Reversed: true,
			},
		},
	}
	all, err := root.CVEDataStore.Search(ctx, q)
	if err != nil {
		return nil, err
	}

	fixable, err := root.CVEDataStore.Count(ctx,
		search.ConjunctionQuery(q, search.NewQueryBuilder().AddBools(search.Fixable, true).ProtoQuery()))
	if err != nil {
		return nil, err
	}

	return &PlottedVulnerabilitiesResolver{
		root:    root,
		all:     search.ResultsToIDs(all),
		fixable: fixable,
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
	q, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}

	if len(pvr.all) == 0 {
		return nil, nil
	}

	cvesInterface, err := paginationWrapper{
		pv: q.GetPagination(),
	}.paginate(pvr.all, nil)
	if err != nil {
		return nil, err
	}

	vulns, err := pvr.root.CVEDataStore.GetBatch(ctx, cvesInterface.([]string))
	if err != nil {
		return nil, err
	}

	vulnerabilityResolvers := make([]VulnerabilityResolver, 0, len(vulns))
	for _, vuln := range vulns {
		vulnerabilityResolvers = append(vulnerabilityResolvers, &cVEResolver{root: pvr.root, data: vuln})
	}
	return vulnerabilityResolvers, nil
}
