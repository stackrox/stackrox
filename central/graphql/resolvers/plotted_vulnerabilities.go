package resolvers

import (
	"context"

	"github.com/stackrox/rox/pkg/search"
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
	fixable []string
}

func newPlottedVulnerabilitiesResolver(ctx context.Context, root *Resolver, args RawQuery) (*PlottedVulnerabilitiesResolver, error) {
	q, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}

	q = tryUnsuppressedQuery(q)
	all, err := root.CVEDataStore.Search(ctx, q)
	if err != nil {
		return nil, err
	}

	fixable, err := root.CVEDataStore.Search(ctx,
		search.NewConjunctionQuery(q, search.NewQueryBuilder().AddBools(search.Fixable, true).ProtoQuery()))
	if err != nil {
		return nil, err
	}

	return &PlottedVulnerabilitiesResolver{
		root:    root,
		all:     search.ResultsToIDs(all),
		fixable: search.ResultsToIDs(fixable),
	}, nil
}

// BasicVulnCounter returns the vulnCounter for scatter-plot with only total and fixable
func (pvr *PlottedVulnerabilitiesResolver) BasicVulnCounter(ctx context.Context) (*VulnerabilityCounterResolver, error) {
	return &VulnerabilityCounterResolver{
		all: &VulnerabilityFixableCounterResolver{
			total:   int32(len(pvr.all)),
			fixable: int32(len(pvr.fixable)),
		},
	}, nil
}

// Vulns returns the vulns for scatter-plot
func (pvr *PlottedVulnerabilitiesResolver) Vulns(ctx context.Context, args PaginatedQuery) ([]VulnerabilityResolver, error) {
	q, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}

	pagination := q.GetPagination()
	q = search.NewQueryBuilder().AddDocIDs(pvr.all...).ProtoQuery()
	q.Pagination = pagination

	paginatedVulns, err := pvr.root.CVEDataStore.SearchRawCVEs(ctx, q)
	if err != nil {
		return nil, err
	}

	vulns := make([]VulnerabilityResolver, 0, len(paginatedVulns))
	for _, vuln := range paginatedVulns {
		vulns = append(vulns, &cVEResolver{root: pvr.root, data: vuln})
	}
	return vulns, nil
}
