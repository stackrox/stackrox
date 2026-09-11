import { useCallback } from 'react';
import { fetchDeploymentsCountByQuery } from 'services/DeploymentsService';
import useRestQuery from 'hooks/useRestQuery';

/**
 * Fetch deployment count via GET /v1/deploymentscount using a raw query string.
 * Same RawQuery format as GraphQL deploymentCount.
 */
export default function useFetchDeploymentCountByQuery(query = '') {
    const restQuery = useCallback(() => fetchDeploymentsCountByQuery(query), [query]);

    return useRestQuery(restQuery);
}
