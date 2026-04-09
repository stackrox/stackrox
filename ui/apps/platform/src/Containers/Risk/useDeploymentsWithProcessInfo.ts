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
    showDeleted: boolean;
};

export default function useDeploymentsWithProcessInfo({
    searchFilter,
    sortOption,
    page,
    perPage,
    showDeleted,
}: UseDeploymentsWithProcessInfoParams): UseRestQueryReturn<ListDeploymentWithProcessInfo[]> {
    const shouldHideOrchestratorComponents =
        localStorage.getItem(ORCHESTRATOR_COMPONENTS_KEY) !== 'true';

    const requestFn = useCallback(() => {
        const effectiveSearchFilter = {
            ...searchFilter,
            ...(shouldHideOrchestratorComponents ? { 'Orchestrator Component': 'false' } : {}),
            // When showing deleted deployments, add the tombstone filter so the backend
            // bypasses its default active-only exclusion and returns soft-deleted records.
            ...(showDeleted ? { 'Tombstone Deleted At': '*' } : {}),
        };

        return fetchDeploymentsWithProcessInfo(effectiveSearchFilter, sortOption, page, perPage);
    }, [searchFilter, sortOption, page, perPage, shouldHideOrchestratorComponents, showDeleted]);

    return useRestQuery(requestFn);
}
