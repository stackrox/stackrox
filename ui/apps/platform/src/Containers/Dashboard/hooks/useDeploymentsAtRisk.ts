import { useCallback } from 'react';
import { fetchDeploymentsWithProcessInfo } from 'services/DeploymentsService';
import { SearchFilter } from 'types/search';
import useRestQuery from 'hooks/useRestQuery';

export default function useDeploymentsAtRisk(searchFilter: SearchFilter, numberOfResults = 6) {
    const restQuery = useCallback(() => {
        const { request, cancel } = fetchDeploymentsWithProcessInfo(
            searchFilter,
            { field: 'Deployment Risk Priority', reversed: 'false' },
            0,
            numberOfResults
        );

        return {
            request: request.then((results) => results.map(({ deployment }) => deployment)),
            cancel,
        };
    }, [searchFilter, numberOfResults]);

    return useRestQuery(restQuery);
}
