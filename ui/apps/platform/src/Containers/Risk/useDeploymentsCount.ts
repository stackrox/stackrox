import { useCallback } from 'react';

import useRestQuery from 'hooks/useRestQuery';
import type { UseRestQueryReturn } from 'hooks/useRestQuery';
import { fetchDeploymentsCount } from 'services/DeploymentsService';
import type { SearchFilter } from 'types/search';
import { ORCHESTRATOR_COMPONENTS_KEY } from 'utils/orchestratorComponents';

type UseDeploymentsCountParams = {
    searchFilter: SearchFilter;
};

export default function useDeploymentsCount({
    searchFilter,
}: UseDeploymentsCountParams): UseRestQueryReturn<number> {
    const shouldHideOrchestratorComponents =
        localStorage.getItem(ORCHESTRATOR_COMPONENTS_KEY) !== 'true';

    const requestFn = useCallback(() => {
        const effectiveSearchFilter = {
            ...searchFilter,
            ...(shouldHideOrchestratorComponents ? { 'Orchestrator Component': 'false' } : {}),
        };

        return fetchDeploymentsCount(effectiveSearchFilter);
    }, [searchFilter, shouldHideOrchestratorComponents]);

    return useRestQuery(requestFn);
}
