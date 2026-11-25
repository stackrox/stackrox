import { useCallback, useEffect } from 'react';

import useRestQuery from 'hooks/useRestQuery';
import { fetchVulnMgmtCounts } from 'services/VulnMgmtService';
import type { VulnMgmtCounts } from 'services/VulnMgmtService';

import { useVulnMgmtCacheVersion } from './useVulnMgmtCacheVersion';

export function useVulnMgmtCounts(query: string): {
    counts: VulnMgmtCounts | undefined;
    loading: boolean;
    error: Error | undefined;
    refetch: () => void;
} {
    const cacheVersion = useVulnMgmtCacheVersion();
    const requestFn = useCallback(() => fetchVulnMgmtCounts(query), [query]);
    const { data, isLoading, error, refetch } = useRestQuery(requestFn);

    // Refetch when a mutation invalidates the REST cache.
    useEffect(() => {
        if (cacheVersion > 0) {
            refetch();
        }
    }, [cacheVersion, refetch]);

    return { counts: data, loading: isLoading, error: error ?? undefined, refetch };
}
