package resolvers

import (
	"context"
	"errors"
)

func getPlottedVulnsIdsAndFixableCount(ctx context.Context, root *Resolver, args RawQuery) ([]string, int, error) {
	return nil, 0, errors.New("unexpected access to legacy CVE datastore")
}

func unwrappedPlottedVulnerabilities(ctx context.Context, resolver *Resolver, cveIds []string, args PaginatedQuery) ([]*cVEResolver, error) {
	q, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}

	if len(cveIds) == 0 {
		return nil, nil
	}

	cves, err := paginate(q.GetPagination(), cveIds, nil)
	if err != nil {
		return nil, err
	}

	vulns, err := resolver.CVEDataStore.GetBatch(ctx, cves)
	if err != nil {
		return nil, err
	}

	vulnResolvers, err := resolver.wrapCVEs(vulns, err)
	return vulnResolvers, err
}
