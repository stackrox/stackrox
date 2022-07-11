package resolvers

import (
	"context"

	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
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

func newPlottedNodeVulnerabilitiesResolver(ctx context.Context, root *Resolver, args RawQuery) (*PlottedNodeVulnerabilitiesResolver, error) {
	if !features.PostgresDatastore.Enabled() {
		q := withNodeCveTypeFiltering(args.String())
		allCveIds, fixableCount, err := getPlottedVulnsIdsAndFixableCount(ctx, root, RawQuery{Query: &q})
		if err != nil {
			return nil, err
		}

		return &PlottedNodeVulnerabilitiesResolver{
			root:    root,
			all:     allCveIds,
			fixable: fixableCount,
		}, nil
	}

	query, err := getPlottedVulnsV1Query(args)
	if err != nil {
		return nil, err
	}
	vulnLoader, err := loaders.GetNodeCVELoader(ctx)
	if err != nil {
		return nil, err
	}
	allCves, err := vulnLoader.FromQuery(ctx, query)
	if err != nil {
		return nil, err
	}
	allCveIds := make([]string, 0, len(allCves))
	for _, cve := range allCves {
		allCveIds = append(allCveIds, cve.GetId())
	}

	fixableQuery, err := getPlottedVulnsV1Query(args, search.ExcludeFieldLabel(search.Fixable))
	fixableCount, err := vulnLoader.CountFromQuery(ctx,
		search.ConjunctionQuery(fixableQuery, search.NewQueryBuilder().AddBools(search.Fixable, true).ProtoQuery()))
	if err != nil {
		return nil, err
	}

	return &PlottedNodeVulnerabilitiesResolver{
		root:    root,
		all:     allCveIds,
		fixable: int(fixableCount),
	}, nil
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
func (pvr *PlottedNodeVulnerabilitiesResolver) NodeVulnerabilities(ctx context.Context, args PaginatedQuery) ([]NodeVulnerabilityResolver, error) {
	if !features.PostgresDatastore.Enabled() {
		vulnResolvers, err := unwrappedPlottedVulnerabilities(ctx, pvr.root, pvr.all, PaginatedQuery{Pagination: args.Pagination})
		if err != nil {
			return nil, err
		}

		ret := make([]NodeVulnerabilityResolver, 0, len(vulnResolvers))
		for _, resolver := range vulnResolvers {
			ret = append(ret, resolver)
		}
		return ret, nil
	}

	if len(pvr.all) == 0 {
		return nil, nil
	}
	q := search.NewQueryBuilder().AddExactMatches(search.CVEID, pvr.all...).Query()
	return pvr.root.NodeVulnerabilities(ctx, PaginatedQuery{Query: &q, Pagination: args.Pagination})
}
