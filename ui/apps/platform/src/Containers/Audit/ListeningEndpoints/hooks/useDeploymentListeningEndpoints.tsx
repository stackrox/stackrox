import { useCallback } from 'react';
import { UsePaginatedQueryReturn, usePaginatedQuery } from 'hooks/usePaginatedQuery';
import { listDeployments } from 'services/DeploymentsService';
import {
    ProcessListeningOnPort,
    getListeningEndpointsForDeployment,
} from 'services/ProcessListeningOnPortsService';
import { ListDeployment } from 'types/deployment.proto';

const sortOptions = { field: 'Deployment', reversed: 'false' };
const pageSize = 10;

/**
 * Returns a paginated list of deployments with their listening endpoints.
 */
export function useDeploymentListeningEndpoints(): UsePaginatedQueryReturn<
    ListDeployment & { listeningEndpoints: ProcessListeningOnPort[] }
> {
    const queryFn = useCallback((page: number) => {
        return listDeployments({}, sortOptions, page, pageSize).then((res) => {
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
    }, []);

    return usePaginatedQuery(queryFn, pageSize);
}
