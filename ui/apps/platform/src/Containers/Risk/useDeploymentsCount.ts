import { useCallback } from 'react';

import useFeatureFlags from 'hooks/useFeatureFlags';
import useRestQuery from 'hooks/useRestQuery';
import type { UseRestQueryReturn } from 'hooks/useRestQuery';
import { fetchDeploymentsCount } from 'services/DeploymentsService';
import type { SearchFilter } from 'types/search';
import { withActiveDeploymentFilter } from 'utils/deploymentUtils';

type UseDeploymentsCountParams = {
    searchFilter: SearchFilter;
};

export default function useDeploymentsCount({
    searchFilter,
}: UseDeploymentsCountParams): UseRestQueryReturn<number> {
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const isDeploymentSoftDeletionEnabled = isFeatureFlagEnabled('ROX_DEPLOYMENT_SOFT_DELETION');

    const requestFn = useCallback(() => {
        const effectiveSearchFilter = withActiveDeploymentFilter(
            {
                ...searchFilter,
            },
            isDeploymentSoftDeletionEnabled
        );

        return fetchDeploymentsCount(effectiveSearchFilter);
    }, [searchFilter, isDeploymentSoftDeletionEnabled]);

    return useRestQuery(requestFn);
}
