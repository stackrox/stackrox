import { useCallback } from 'react';
import { listDeployments } from 'services/DeploymentsService';
import { getListeningEndpointsForDeployment } from 'services/ProcessListeningOnPortsService';
import useFeatureFlags from 'hooks/useFeatureFlags';
import useRestQuery from 'hooks/useRestQuery';
import type { ApiSortOption, SearchFilter } from 'types/search';
import { withActiveDeploymentFilter } from 'utils/deploymentUtils';

/**
 * Returns a paginated list of deployments with their listening endpoints.
 */
export function useDeploymentListeningEndpoints(
    searchFilter: SearchFilter,
    sortOption: ApiSortOption,
    page: number,
    perPage: number
) {
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const isDeploymentSoftDeletionEnabled = isFeatureFlagEnabled('ROX_DEPLOYMENT_SOFT_DELETION');

    const queryFn = useCallback(() => {
        const effectiveSearchFilter = withActiveDeploymentFilter(
            searchFilter,
            isDeploymentSoftDeletionEnabled
        );
        return listDeployments(effectiveSearchFilter, sortOption, page, perPage).then((res) => {
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
    }, [searchFilter, sortOption, page, perPage, isDeploymentSoftDeletionEnabled]);

    return useRestQuery(queryFn);
}
