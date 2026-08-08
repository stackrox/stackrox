import { useCallback } from 'react';
import { listDeployments } from 'services/DeploymentsService';
import { getListeningEndpointsForDeployment } from 'services/ProcessListeningOnPortsService';
import useRestQuery from 'hooks/useRestQuery';
import type { ApiSortOption, SearchFilter } from 'types/search';

export const DEFAULT_ENDPOINTS_PER_PAGE = 20;

/**
 * Returns a paginated list of deployments with the first page of their listening endpoints.
 */
export function useDeploymentListeningEndpoints(
    searchFilter: SearchFilter,
    sortOption: ApiSortOption,
    page: number,
    perPage: number
) {
    const queryFn = useCallback(() => {
        return listDeployments(searchFilter, sortOption, page, perPage).then((res) => {
            return Promise.all(
                res.map((deployment) => {
                    const { request } = getListeningEndpointsForDeployment(deployment.id, {
                        offset: 0,
                        limit: DEFAULT_ENDPOINTS_PER_PAGE,
                    });
                    return request.then((response) => ({
                        ...deployment,
                        listeningEndpoints: response.listeningEndpoints,
                        totalListeningEndpoints: response.totalListeningEndpoints,
                    }));
                })
            );
        });
    }, [searchFilter, sortOption, page, perPage]);

    return useRestQuery(queryFn);
}
