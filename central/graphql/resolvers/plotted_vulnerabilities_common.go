package resolvers

import (
	"context"

	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/pkg/search"
)

func getPlottedVulnsIdsAndFixableCount(ctx context.Context, root *Resolver, args RawQuery) ([]string, int, error) {
	q, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, 0, err
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
		return nil, 0, err
	}

	fixable, err := root.CVEDataStore.Count(ctx,
		search.ConjunctionQuery(q, search.NewQueryBuilder().AddBools(search.Fixable, true).ProtoQuery()))
	if err != nil {
		return nil, 0, err
	}

	return search.ResultsToIDs(all), fixable, nil
}

func unwrappedPlottedVulnerabilities(ctx context.Context, resolver *Resolver, cveIds []string, args PaginatedQuery) ([]*cVEResolver, error) {
	q, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}

	if len(cveIds) == 0 {
		return nil, nil
	}

	cvesInterface, err := paginationWrapper{
		pv: q.GetPagination(),
	}.paginate(cveIds, nil)
	if err != nil {
		return nil, err
	}

	vulns, err := resolver.CVEDataStore.GetBatch(ctx, cvesInterface.([]string))
	if err != nil {
		return nil, err
	}

	vulnResolvers, err := resolver.wrapCVEs(vulns, err)
	return vulnResolvers, err
}
