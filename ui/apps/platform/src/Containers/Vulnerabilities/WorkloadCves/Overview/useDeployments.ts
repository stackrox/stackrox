import { useQuery } from '@apollo/client';
import type { QueryHookOptions } from '@apollo/client';

import { getPaginationParams } from 'utils/searchUtils';
import type { ApiSortOption } from 'types/search';
import type useURLPagination from 'hooks/useURLPagination';
import useFeatureFlags from 'hooks/useFeatureFlags';
import {
    deploymentListQuery,
    simplifiedDeploymentListQuery,
} from '../Tables/DeploymentOverviewTable';
import type { Deployment } from '../Tables/DeploymentOverviewTable';

export function useDeployments({
    query,
    pagination,
    sortOption,
    options = {},
}: {
    query: string;
    pagination: ReturnType<typeof useURLPagination>;
    sortOption: ApiSortOption | undefined;
    options?: Omit<QueryHookOptions<{ deployments: Deployment[] }>, 'variables'>;
}) {
    const { page, perPage } = pagination;
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const isSimplifiedSeverity = isFeatureFlagEnabled('ROX_VULN_MGMT_UNIFIED_CVE_VIEW');
    const gqlQuery = isSimplifiedSeverity ? simplifiedDeploymentListQuery : deploymentListQuery;

    return useQuery<{
        deployments: Deployment[];
    }>(gqlQuery, {
        variables: {
            query,
            pagination: getPaginationParams({ page, perPage, sortOption }),
        },
        ...options,
    });
}
