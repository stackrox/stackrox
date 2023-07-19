import { useCallback } from 'react';
import { listDeployments } from 'services/DeploymentsService';
import { getListeningEndpointsForDeployment } from 'services/ProcessListeningOnPortsService';
import useRestQuery from 'hooks/useRestQuery';

const sortOptions = { field: 'Deployment', reversed: 'false' };

/**
 * Returns a paginated list of deployments with their listening endpoints.
 */
export function useDeploymentListeningEndpoints(page, perPage) {
    const queryFn = useCallback(() => {
        return listDeployments({}, sortOptions, page - 1, perPage).then((res) => {
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
    }, [page, perPage]);

    return useRestQuery(queryFn);
}
