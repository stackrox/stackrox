import { useCallback } from 'react';

import type useURLPagination from 'hooks/useURLPagination';
import useRestQuery from 'hooks/useRestQuery';
import { fetchCVEs } from 'services/VulnMgmtService';
import type { ApiSortOption } from 'types/search';

import type { WorkloadCVEListItem } from '../Tables/WorkloadCVEOverviewTable';
import { useVulnMgmtCacheVersion } from './useVulnMgmtCacheVersion';

export function useVulnMgmtCVEs({
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
        () => fetchCVEs({ query, page, perPage, sortOption }),
        // cacheVersion is included so invalidation triggers a refetch.
        [query, page, perPage, sortOption, cacheVersion]
    );

    const { data, isLoading, error, refetch } = useRestQuery(requestFn);

    return {
        loading: isLoading,
        error: error ?? undefined,
        data: data
            ? {
                  imageCVEs: data.items.map(
                      (item): WorkloadCVEListItem => ({
                          cve: item.cve,
                          topCVSS: item.topCvss,
                          topNvdCVSS: item.topNvdCvss,
                          topEpssProbability: item.topEpssProbability,
                          affectedImageCount: item.affectedImageCount,
                          firstDiscoveredInSystem: item.firstDiscoveredInSystem ?? null,
                          publishedOn: item.publishedOn ?? null,
                          pendingExceptionCount: item.pendingExceptionCount,
                          knownExploit: item.knownExploit,
                          knownRansomwareCampaignUse: item.knownRansomwareCampaignUse,
                      })
                  ),
                  totalCount: data.totalCount,
              }
            : undefined,
        refetch,
    };
}
