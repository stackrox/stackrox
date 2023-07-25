import { useCallback } from 'react';
import { listDeployments } from 'services/DeploymentsService';
import { getListeningEndpointsForDeployment } from 'services/ProcessListeningOnPortsService';
import useRestQuery from 'hooks/useRestQuery';
import { ApiSortOption, SearchFilter } from 'types/search';

/**
 * Returns a paginated list of deployments with their listening endpoints.
 */
export function useDeploymentListeningEndpoints(
    searchFilter: SearchFilter,
    sortOption: ApiSortOption,
    page: number,
    perPage: number
) {
    const queryFn = useCallback(() => {
        return listDeployments(searchFilter, sortOption, page - 1, perPage).then((res) => {
            return Promise.all(
                res.map((deployment) => {
                    const { request } = getListeningEndpointsForDeployment(deployment.id);
                    return request.then((listeningEndpoints) => ({
                        ...deployment,
                        listeningEndpoints,
                    }));
                })
            );
        });
    }, [searchFilter, sortOption, page, perPage]);

    return useRestQuery(queryFn);
}
