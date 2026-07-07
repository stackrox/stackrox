import axios from './instance';

import type { ApiSortOption } from 'types/search';
import { getPaginationParams } from 'utils/searchUtils';

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
    const pagination = getPaginationParams({ page, perPage, sortOption });
    if (pagination.offset) {
        params.set('pagination.offset', String(pagination.offset));
    }
    if (pagination.limit) {
        params.set('pagination.limit', String(pagination.limit));
    }
    if (pagination.sortOption) {
        params.set('pagination.sortOption.field', pagination.sortOption.field);
        if (pagination.sortOption.reversed) {
            params.set('pagination.sortOption.reversed', 'true');
        }
    }

    return axios
        .get<CVEListResponse>(`/api/v2/vulnmgmt/cves?${params.toString()}`)
        .then((response) => response.data);
}

export function fetchCVEDetail(cve: string): Promise<CVEDetailResponse> {
    return axios
        .get<CVEDetailResponse>(`/api/v2/vulnmgmt/cves/${encodeURIComponent(cve)}/detail`)
        .then((response) => response.data);
}
