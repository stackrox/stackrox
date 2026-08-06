import axios from './instance';

import type { ApiSortOption, ApiSortOptionSingle } from 'types/search';

export type CVEListItem = {
    cve: string;
    topSeverity: string;
    topCvss: number;
    topNvdCvss: number;
    topEpssProbability: number;
    affectedImageCount: number;
    firstDiscoveredInSystem: string;
    publishedOn: string;
    pendingExceptionCount: number;
};

export type CVEListResponse = {
    items: CVEListItem[];
    totalCount: number;
};

export type CVEDetailItem = {
    cve: string;
    summary: string;
    link: string;
    operatingSystem: string;
};

export type CVEDetailResponse = {
    distroDetails: CVEDetailItem[];
};

function appendSortParams(params: URLSearchParams, sortOption: ApiSortOption | undefined) {
    if (!sortOption) {
        return;
    }

    const options: ApiSortOptionSingle[] = Array.isArray(sortOption) ? sortOption : [sortOption];

    if (options.length > 0) {
        const opt = options[0];
        params.set('pagination.sortOption.field', opt.field);
        if (opt.reversed) {
            params.set('pagination.sortOption.reversed', 'true');
        }
        if (opt.aggregateBy) {
            params.set(
                'pagination.sortOption.aggregateBy.aggregateFunc',
                opt.aggregateBy.aggregateFunc
            );
            if (opt.aggregateBy.distinct) {
                params.set('pagination.sortOption.aggregateBy.distinct', 'true');
            }
        }
    }
}

export function fetchCVEList({
    query,
    page,
    perPage,
    sortOption,
}: {
    query: string;
    page: number;
    perPage: number;
    sortOption: ApiSortOption | undefined;
}): Promise<CVEListResponse> {
    const params = new URLSearchParams();
    if (query) {
        params.set('query', query);
    }

    const safePage = Math.max(1, page);
    const safePerPage = Math.max(0, perPage);
    const offset = (safePage - 1) * safePerPage;

    if (offset) {
        params.set('pagination.offset', String(offset));
    }
    if (safePerPage) {
        params.set('pagination.limit', String(safePerPage));
    }
    appendSortParams(params, sortOption);

    return axios
        .get<CVEListResponse>(`/api/v2/vulnmgmt/cves?${params.toString()}`, {
            timeout: 180000,
        })
        .then((response) => response.data);
}

export function fetchCVEDetail(cve: string): Promise<CVEDetailResponse> {
    return axios
        .get<CVEDetailResponse>(`/api/v2/vulnmgmt/cves/${encodeURIComponent(cve)}/detail`)
        .then((response) => response.data);
}
