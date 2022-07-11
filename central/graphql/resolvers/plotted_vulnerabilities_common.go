package resolvers

import (
	"context"
	"errors"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/search"
)

func getPlottedVulnsIdsAndFixableCount(ctx context.Context, root *Resolver, args RawQuery) ([]string, int, error) {
	if features.PostgresDatastore.Enabled() {
		return nil, 0, errors.New("unexpected access to legacy CVE datastore")
	}
	query, err := getPlottedVulnsV1Query(args)
	if err != nil {
		return nil, 0, err
	}
	all, err := root.CVEDataStore.Search(ctx, query)
	if err != nil {
		return nil, 0, err
	}

	fixable, err := root.CVEDataStore.Count(ctx,
		search.ConjunctionQuery(query, search.NewQueryBuilder().AddBools(search.Fixable, true).ProtoQuery()))
	if err != nil {
		return nil, 0, err
	}

	return search.ResultsToIDs(all), fixable, nil
}

func getPlottedVulnsV1Query(args RawQuery, opts ...search.ParseQueryOption) (*v1.Query, error) {
	q, err := args.AsV1QueryOrEmpty(opts...)
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
	return q, nil
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
