package resolvers

import (
	"context"

	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddType("PlottedNodeVulnerabilities", []string{
			"basicNodeVulnerabilityCounter: VulnerabilityCounter!",
			"nodeVulnerabilities(pagination: Pagination): [NodeVulnerability]!",
		}),
	)
}

// PlottedNodeVulnerabilitiesResolver returns the data required by top risky nodes scatter-plot on vuln mgmt dashboard
type PlottedNodeVulnerabilitiesResolver struct {
	root    *Resolver
	all     []string
	fixable int
}

func (resolver *Resolver) wrapPlottedNodeVulnerabilities(all []string, fixable int) (*PlottedNodeVulnerabilitiesResolver, error) {
	return &PlottedNodeVulnerabilitiesResolver{
		root:    resolver,
		all:     all,
		fixable: fixable,
	}, nil
}

// PlottedNodeVulnerabilities - returns node vulns
func (resolver *Resolver) PlottedNodeVulnerabilities(ctx context.Context, args RawQuery) (*PlottedNodeVulnerabilitiesResolver, error) {
	if !features.PostgresDatastore.Enabled() {
		q := withNodeCveTypeFiltering(args.String())
		allCveIds, fixableCount, err := getPlottedVulnsIdsAndFixableCount(ctx, resolver, RawQuery{Query: &q})
		if err != nil {
			return nil, err
		}

		return resolver.wrapPlottedNodeVulnerabilities(allCveIds, fixableCount)
	}

	query, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}
	logErrorOnQueryContainingField(query, search.Fixable, "PlottedNodeVulnerabilities")

	query.Pagination = &v1.QueryPagination{
		SortOptions: []*v1.QuerySortOption{
			{
				Field:    search.CVSS.String(),
				Reversed: true,
			},
		},
	}
	query = tryUnsuppressedQuery(query)

	vulnLoader, err := loaders.GetNodeCVELoader(ctx)
	if err != nil {
		return nil, err
	}
	allCveIds, err := vulnLoader.GetIDs(ctx, query)
	if err != nil {
		return nil, err
	}

	fixableCount, err := vulnLoader.CountFromQuery(ctx,
		search.ConjunctionQuery(query, search.NewQueryBuilder().AddBools(search.Fixable, true).ProtoQuery()))
	if err != nil {
		return nil, err
	}

	return resolver.wrapPlottedNodeVulnerabilities(allCveIds, int(fixableCount))
}

// BasicNodeVulnerabilityCounter returns the NodeVulnerabilityCounter for scatter-plot with only total and fixable
func (pvr *PlottedNodeVulnerabilitiesResolver) BasicNodeVulnerabilityCounter(_ context.Context) (*VulnerabilityCounterResolver, error) {
	return &VulnerabilityCounterResolver{
		all: &VulnerabilityFixableCounterResolver{
			total:   int32(len(pvr.all)),
			fixable: int32(pvr.fixable),
		},
	}, nil
}

// NodeVulnerabilities returns the node vulnerabilities for top risky nodes scatter-plot
func (pvr *PlottedNodeVulnerabilitiesResolver) NodeVulnerabilities(ctx context.Context, args PaginationWrapper) ([]NodeVulnerabilityResolver, error) {
	if len(pvr.all) == 0 {
		return nil, nil
	}
	q := search.NewQueryBuilder().AddExactMatches(search.CVEID, pvr.all...).Query()
	return pvr.root.NodeVulnerabilities(ctx, PaginatedQuery{Query: &q, Pagination: args.Pagination})
}
