package resolvers

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddType("PlottedNodeVulnerabilities", []string{
			"basicVulnCounter: VulnerabilityCounter!",
			"vulnerabilities(pagination: Pagination): [NodeVulnerability]!",
		}),
	)
}

// PlottedNodeVulnerabilitiesResolver returns the data required by top risky nodes scatter-plot on vuln mgmt dashboard
type PlottedNodeVulnerabilitiesResolver struct {
	root    *Resolver
	all     []string
	fixable int
}

func newPlottedNodeVulnsResolver(ctx context.Context, root *Resolver, args RawQuery) (*PlottedNodeVulnerabilitiesResolver, error) {
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

	return &PlottedNodeVulnerabilitiesResolver{
		root:    root,
		all:     search.ResultsToIDs(all),
		fixable: fixable,
	}, nil
}
