import { useQuery } from '@apollo/client';
import type { QueryHookOptions } from '@apollo/client';

import { getPaginationParams } from 'utils/searchUtils';
import type { ApiSortOption } from 'types/search';
import type useURLPagination from 'hooks/useURLPagination';
import { deploymentListQuery } from '../Tables/DeploymentOverviewTable';
import type { Deployment } from '../Tables/DeploymentOverviewTable';

export function useDeploymentList({
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

    return useQuery<{
        deployments: Deployment[];
    }>(deploymentListQuery, {
        variables: {
            query,
            pagination: getPaginationParams({ page, perPage, sortOption }),
        },
        ...options,
    });
}
