import { useCallback, useEffect } from 'react';

import type useURLPagination from 'hooks/useURLPagination';
import useRestQuery from 'hooks/useRestQuery';
import { fetchImages } from 'services/VulnMgmtService';
import type { ApiSortOption } from 'types/search';

import { useVulnMgmtCacheVersion } from './useVulnMgmtCacheVersion';

export function useVulnMgmtImages({
    query,
    pagination,
    sortOption,
}: {
    query: string;
    pagination: ReturnType<typeof useURLPagination>;
    sortOption: ApiSortOption | undefined;
}) {
    const { page, perPage } = pagination;
    const cacheVersion = useVulnMgmtCacheVersion();
    const requestFn = useCallback(
        () => fetchImages({ query, page, perPage, sortOption }),
        [query, page, perPage, sortOption]
    );

    const { data, isLoading, error, refetch } = useRestQuery(requestFn);

    // Refetch when a mutation invalidates the REST cache.
    useEffect(() => {
        if (cacheVersion > 0) {
            refetch();
        }
    }, [cacheVersion, refetch]);

    return {
        loading: isLoading,
        error: error ?? undefined,
        data: data ? { images: data.items } : undefined,
        refetch,
    };
}
