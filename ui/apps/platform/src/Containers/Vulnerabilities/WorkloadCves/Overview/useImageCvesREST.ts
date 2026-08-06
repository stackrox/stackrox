import { useCallback } from 'react';

import useRestQuery from 'hooks/useRestQuery';
import type useURLPagination from 'hooks/useURLPagination';
import type { ApiSortOption } from 'types/search';

import { fetchCVEList } from 'services/VulnMgmtService';
import type { CVEListResponse } from 'services/VulnMgmtService';
import type { ImageCVE } from '../Tables/WorkloadCVEOverviewTable';

function toCVEListItems(data: CVEListResponse | undefined): ImageCVE[] | undefined {
    if (!data) {
        return undefined;
    }
    return data.items.map((item) => ({
        cve: item.cve,
        topSeverity: item.topSeverity,
        topEpssProbability: item.topEpssProbability,
        topCVSS: item.topCvss,
        topNvdCVSS: item.topNvdCvss,
        affectedImageCount: item.affectedImageCount,
        firstDiscoveredInSystem: item.firstDiscoveredInSystem || null,
        publishedOn: item.publishedOn || null,
        pendingExceptionCount: item.pendingExceptionCount,
    }));
}

export function useImageCvesREST({
    query,
    pagination,
    sortOption,
}: {
    query: string;
    pagination: ReturnType<typeof useURLPagination>;
    sortOption: ApiSortOption | undefined;
}) {
    const { page, perPage } = pagination;

    const requestFn = useCallback(
        () => fetchCVEList({ query, page, perPage, sortOption }),
        [query, page, perPage, sortOption]
    );

    const { data: rawData, isLoading, error } = useRestQuery(requestFn);

    return {
        loading: isLoading,
        error: error ?? undefined,
        data: rawData
            ? { imageCVEs: toCVEListItems(rawData) ?? [], totalCount: rawData.totalCount }
            : undefined,
    };
}
