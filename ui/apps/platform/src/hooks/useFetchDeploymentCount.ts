import { useCallback } from 'react';
import { fetchDeploymentsCount } from 'services/DeploymentsService';
import type { SearchFilter } from 'types/search';
import { getRequestQueryStringForSearchFilter } from 'utils/searchUtils';
import useRestQuery from 'hooks/useRestQuery';

export default function useFetchDeploymentCount(searchFilter: SearchFilter) {
    // Serialize the filter so a new object with the same criteria does not refetch.
    const query = getRequestQueryStringForSearchFilter(searchFilter);
    const restQuery = useCallback(
        () => fetchDeploymentsCount(searchFilter),
        // searchFilter is represented by `query`, which is what the REST API uses.
        // eslint-disable-next-line react-hooks/exhaustive-deps
        [query]
    );

    return useRestQuery(restQuery);
}
