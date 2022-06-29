package resolvers

import (
	"context"

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
	all, err := root.NodeCVEDataStore.Search(ctx, query)
	if err != nil {
		return nil, err
	}
	allCveIds := search.ResultsToIDs(all)

	fixableCount, err := root.NodeCVEDataStore.Count(ctx,
		search.ConjunctionQuery(query, search.NewQueryBuilder().AddBools(search.Fixable, true).ProtoQuery()))
	if err != nil {
		return nil, err
	}

	return &PlottedNodeVulnerabilitiesResolver{
		root:    root,
		all:     allCveIds,
		fixable: fixableCount,
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
		vulnResolvers, err := unwrappedPlottedVulnerabilities(ctx, pvr.root, pvr.all, args)
		if err != nil {
			return nil, err
		}

		ret := make([]NodeVulnerabilityResolver, 0, len(vulnResolvers))
		for _, resolver := range vulnResolvers {
			ret = append(ret, resolver)
		}
		return ret, nil
	}

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

	vulns, err := pvr.root.NodeCVEDataStore.GetBatch(ctx, cvesInterface.([]string))
	if err != nil {
		return nil, err
	}

	vulnResolvers, err := pvr.root.wrapNodeCVEs(vulns, err)
	if err != nil {
		return nil, err
	}

	ret := make([]NodeVulnerabilityResolver, 0, len(vulnResolvers))
	for _, resolver := range vulnResolvers {
		ret = append(ret, resolver)
	}
	return ret, nil
}
