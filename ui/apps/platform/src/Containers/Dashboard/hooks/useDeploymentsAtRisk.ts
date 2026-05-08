import { useCallback } from 'react';
import { fetchDeploymentsWithProcessInfo } from 'services/DeploymentsService';
import type { SearchFilter } from 'types/search';
import useFeatureFlags from 'hooks/useFeatureFlags';
import useRestQuery from 'hooks/useRestQuery';
import { withActiveDeploymentFilter } from 'utils/deploymentUtils';

export default function useDeploymentsAtRisk(searchFilter: SearchFilter, numberOfResults = 6) {
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const isDeploymentSoftDeletionEnabled = isFeatureFlagEnabled('ROX_DEPLOYMENT_SOFT_DELETION');

    const restQuery = useCallback(() => {
        const effectiveSearchFilter = withActiveDeploymentFilter(
            searchFilter,
            isDeploymentSoftDeletionEnabled
        );
        const { request, cancel } = fetchDeploymentsWithProcessInfo(
            effectiveSearchFilter,
            { field: 'Deployment Risk Priority', reversed: false },
            0,
            numberOfResults
        );

        return {
            request: request.then((results) => results.map(({ deployment }) => deployment)),
            cancel,
        };
    }, [searchFilter, numberOfResults, isDeploymentSoftDeletionEnabled]);

    return useRestQuery(restQuery);
}
