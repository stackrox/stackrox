import { useCallback } from 'react';

import useRestQuery from 'hooks/useRestQuery';
import type { UseRestQueryReturn } from 'hooks/useRestQuery';
import { fetchDeploymentsWithProcessInfo } from 'services/DeploymentsService';
import type { ListDeploymentWithProcessInfo } from 'services/DeploymentsService';
import type { ApiSortOption, SearchFilter } from 'types/search';

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
    const requestFn = useCallback(() => {
        return fetchDeploymentsWithProcessInfo(searchFilter, sortOption, page, perPage);
    }, [searchFilter, sortOption, page, perPage]);

    return useRestQuery(requestFn);
}
