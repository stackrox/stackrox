import { useCallback } from 'react';

import type useURLPagination from 'hooks/useURLPagination';
import useRestQuery from 'hooks/useRestQuery';
import { fetchImages } from 'services/VulnMgmtService';
import type { VulnMgmtImageListItem } from 'services/VulnMgmtService';
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
        // cacheVersion is included so invalidation triggers a refetch.
        [query, page, perPage, sortOption, cacheVersion]
    );

    const { data, isLoading, error, refetch } = useRestQuery(requestFn);

    return {
        loading: isLoading,
        error: error ?? undefined,
        data: data
            ? { images: data.items as VulnMgmtImageListItem[], totalCount: data.totalCount }
            : undefined,
        refetch,
    };
}
