import { useCallback } from 'react';

import useFeatureFlags from 'hooks/useFeatureFlags';
import useRestQuery from 'hooks/useRestQuery';
import type { UseRestQueryReturn } from 'hooks/useRestQuery';
import { fetchDeploymentsWithProcessInfo } from 'services/DeploymentsService';
import type { ListDeploymentWithProcessInfo } from 'services/DeploymentsService';
import type { ApiSortOption, SearchFilter } from 'types/search';
import { withActiveDeploymentFilter } from 'utils/deploymentUtils';

type UseDeploymentsWithProcessInfoParams = {
    searchFilter: SearchFilter;
    sortOption: ApiSortOption;
    page: number;
    perPage: number;
};

export default function useDeploymentsWithProcessInfo({
    searchFilter,
    sortOption,
    page,
    perPage,
}: UseDeploymentsWithProcessInfoParams): UseRestQueryReturn<ListDeploymentWithProcessInfo[]> {
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const isDeploymentSoftDeletionEnabled = isFeatureFlagEnabled('ROX_DEPLOYMENT_SOFT_DELETION');

    const requestFn = useCallback(() => {
        const effectiveSearchFilter = withActiveDeploymentFilter(
            {
                ...searchFilter,
            },
            isDeploymentSoftDeletionEnabled
        );

        return fetchDeploymentsWithProcessInfo(effectiveSearchFilter, sortOption, page, perPage);
    }, [searchFilter, sortOption, page, perPage, isDeploymentSoftDeletionEnabled]);

    return useRestQuery(requestFn);
}
