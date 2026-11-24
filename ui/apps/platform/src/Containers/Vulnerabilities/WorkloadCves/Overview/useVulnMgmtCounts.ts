import { useCallback } from 'react';

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
    const requestFn = useCallback(
        () => fetchVulnMgmtCounts(query),
        // cacheVersion is included so invalidation triggers a refetch.
        [query, cacheVersion]
    );
    const { data, isLoading, error, refetch } = useRestQuery(requestFn);
    return { counts: data, loading: isLoading, error: error ?? undefined, refetch };
}
