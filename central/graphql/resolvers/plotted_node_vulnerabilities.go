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
	ctx     context.Context
	root    *Resolver
	all     []string
	fixable int
}

func (resolver *Resolver) wrapPlottedNodeVulnerabilitiesWithContext(ctx context.Context, all []string, fixable int) (*PlottedNodeVulnerabilitiesResolver, error) {
	return &PlottedNodeVulnerabilitiesResolver{
		ctx:     ctx,
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

		return resolver.wrapPlottedNodeVulnerabilitiesWithContext(ctx, allCveIds, fixableCount)
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

	return resolver.wrapPlottedNodeVulnerabilitiesWithContext(ctx, allCveIds, int(fixableCount))
}

// BasicNodeVulnerabilityCounter returns the NodeVulnerabilityCounter for scatter-plot with only total and fixable
func (resolver *PlottedNodeVulnerabilitiesResolver) BasicNodeVulnerabilityCounter(_ context.Context) (*VulnerabilityCounterResolver, error) {
	return &VulnerabilityCounterResolver{
		all: &VulnerabilityFixableCounterResolver{
			total:   int32(len(resolver.all)),
			fixable: int32(resolver.fixable),
		},
	}, nil
}

// NodeVulnerabilities returns the node vulnerabilities for top risky nodes scatter-plot
func (resolver *PlottedNodeVulnerabilitiesResolver) NodeVulnerabilities(_ context.Context, args PaginationWrapper) ([]NodeVulnerabilityResolver, error) {
	if len(resolver.all) == 0 {
		return nil, nil
	}
	q := search.NewQueryBuilder().AddExactMatches(search.CVEID, resolver.all...).Query()
	return resolver.root.NodeVulnerabilities(resolver.ctx, PaginatedQuery{Query: &q, Pagination: args.Pagination})
}
