import { useCallback } from 'react';

import useRestQuery from 'hooks/useRestQuery';
import type { UseRestQueryReturn } from 'hooks/useRestQuery';
import { fetchDeploymentsWithProcessInfo } from 'services/DeploymentsService';
import type { ListDeploymentWithProcessInfo } from 'services/DeploymentsService';
import type { ApiSortOption, SearchFilter } from 'types/search';
import { ORCHESTRATOR_COMPONENTS_KEY } from 'utils/orchestratorComponents';

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
    const shouldHideOrchestratorComponents =
        localStorage.getItem(ORCHESTRATOR_COMPONENTS_KEY) !== 'true';

    const requestFn = useCallback(() => {
        const effectiveSearchFilter = {
            ...searchFilter,
            ...(shouldHideOrchestratorComponents ? { 'Orchestrator Component': 'false' } : {}),
        };

        return fetchDeploymentsWithProcessInfo(effectiveSearchFilter, sortOption, page, perPage);
    }, [searchFilter, sortOption, page, perPage, shouldHideOrchestratorComponents]);

    return useRestQuery(requestFn);
}
