import { useCallback } from 'react';

import useRestQuery from 'hooks/useRestQuery';
import type { UseRestQueryReturn } from 'hooks/useRestQuery';
import { fetchDeploymentsCount } from 'services/DeploymentsService';
import type { SearchFilter } from 'types/search';

type UseDeploymentsCountParams = {
    searchFilter: SearchFilter;
};

export default function useDeploymentsCount({
    searchFilter,
}: UseDeploymentsCountParams): UseRestQueryReturn<number> {
    const requestFn = useCallback(() => {
        return fetchDeploymentsCount(searchFilter);
    }, [searchFilter]);

    return useRestQuery(requestFn);
}
