import { useCallback } from 'react';

import useRestQuery from 'hooks/useRestQuery';
import { listDeployments } from 'services/DeploymentsService';
import type { ApiSortOption, SearchFilter } from 'types/search';

type UseWorkloadIdResult = {
    id: string | undefined;
    isLoading: boolean;
    error: Error | undefined;
};

export function useWorkloadId({
    ns,
    name,
}: {
    ns: string | undefined;
    name: string | undefined;
}): UseWorkloadIdResult {
    const deploymentIdQuery = useCallback(() => {
        // Quote search values to ensure exact match instead of prefix match
        const searchFilter: SearchFilter = { Namespace: `"${ns}"`, Deployment: `"${name}"` };
        const sortOption: ApiSortOption = { field: 'Deployment', reversed: false };
        return listDeployments(searchFilter, sortOption, 1, 1);
    }, [ns, name]);

    const { data, isLoading, error } = useRestQuery(deploymentIdQuery);
    const id = data?.[0]?.id;

    if (!ns || !name) {
        return {
            id: undefined,
            isLoading: false,
            error: new Error(
                `An invalid namespace or name was provided. Namespace: ${ns} Name: ${name}`
            ),
        };
    }

    if (!id && !isLoading && !error) {
        return {
            id: undefined,
            isLoading: false,
            error: new Error(`A workload id could not be found. Namespace: ${ns} Name: ${name}`),
        };
    }

    return { id, isLoading, error };
}
