import type { ApiSortOption, ApiSortOptionSingle } from 'types/search';

import axios from './instance';

// Short-lived response cache. It restores the instant back-navigation behavior that the
// Apollo cache provided for the GraphQL list queries. Invalidate explicitly after
// mutations (watch/unwatch image, exception requests, ...).
const CACHE_TTL_MS = 30_000;

type CacheEntry = { data: unknown; timestamp: number };

const responseCache = new Map<string, CacheEntry>();
const invalidationListeners = new Set<() => void>();

export function invalidateVulnMgmtCache(): void {
    responseCache.clear();
    invalidationListeners.forEach((listener) => listener());
}

export function subscribeVulnMgmtCache(listener: () => void): () => void {
    invalidationListeners.add(listener);
    return () => {
        invalidationListeners.delete(listener);
    };
}

function getCached<T>(url: string): Promise<T> {
    const cached = responseCache.get(url);
    if (cached && Date.now() - cached.timestamp < CACHE_TTL_MS) {
        return Promise.resolve(cached.data as T);
    }
    // Some of these queries walk large CVE tables and can legitimately take tens of
    // seconds; the client default is 10s, which aborted the counts request.
    return axios.get<T>(url, { timeout: 180000 }).then((response) => {
        responseCache.set(url, { data: response.data, timestamp: Date.now() });
        return response.data;
    });
}

export type CVESeverityCounts = {
    critical: number;
    important: number;
    moderate: number;
    low: number;
    unknown: number;
};

export type VulnMgmtCounts = {
    cve: number;
    image: number;
    deployment: number;
    severity: CVESeverityCounts;
};

export type CVEListItem = {
    cve: string;
    topCvss: number;
    topNvdCvss: number;
    topEpssProbability: number;
    affectedImageCount: number;
    firstDiscoveredInSystem?: string;
    publishedOn?: string;
    pendingExceptionCount: number;
    knownExploit: boolean;
    knownRansomwareCampaignUse: boolean;
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

export type VulnMgmtImageName = {
    registry: string;
    remote: string;
    tag: string;
    fullName: string;
};

export type VerifiedStatus =
    | 'CORRUPTED_SIGNATURE'
    | 'FAILED_VERIFICATION'
    | 'GENERIC_ERROR'
    | 'INVALID_SIGNATURE_ALGO'
    | 'UNSET'
    | 'VERIFIED';

export type SignatureVerificationResult = {
    description: string | undefined;
    status: VerifiedStatus;
    verificationTime: string | undefined;
    verifiedImageReferences: string[];
    verifierId: string;
    verifierName: string;
};

export type VulnMgmtImageListItem = {
    id: string;
    digest: string;
    name: VulnMgmtImageName;
    operatingSystem: string;
    scanTime?: string;
    createdAt?: string;
    scanNotes: string[];
    notes: string[];
    watchStatus: 'WATCHED' | 'NOT_WATCHED' | 'UNKNOWN';
    deploymentCount: number;
    matchedCveCount: number;
    signatureVerificationData?: { results: SignatureVerificationResult[] } | null;
};

export type ImageListResponse = {
    items: VulnMgmtImageListItem[];
    totalCount: number;
};

export type VulnMgmtDeploymentListItem = {
    id: string;
    name: string;
    type: string;
    cluster: string;
    namespace: string;
    createdAt?: string;
    imageCount: number;
    matchedCveCount: number;
};

export type DeploymentListResponse = {
    items: VulnMgmtDeploymentListItem[];
    totalCount: number;
};

export type FetchListParams = {
    query: string;
    page: number;
    perPage: number;
    sortOption: ApiSortOption | undefined;
};

function appendPaginationAndSort(
    params: URLSearchParams,
    { page, perPage, sortOption }: Omit<FetchListParams, 'query'>
) {
    const safePage = Math.max(1, page);
    const safePerPage = Math.max(0, perPage);
    const offset = (safePage - 1) * safePerPage;
    if (offset) {
        params.set('pagination.offset', String(offset));
    }
    if (safePerPage) {
        params.set('pagination.limit', String(safePerPage));
    }

    const opt: ApiSortOptionSingle | undefined = Array.isArray(sortOption)
        ? sortOption[0]
        : sortOption;
    if (!opt) {
        return;
    }
    params.set('pagination.sortOption.field', opt.field);
    if (opt.reversed) {
        params.set('pagination.sortOption.reversed', 'true');
    }
    if (opt.aggregateBy) {
        params.set(
            'pagination.sortOption.aggregateBy.aggrFunc',
            opt.aggregateBy.aggregateFunc.toUpperCase()
        );
        if (opt.aggregateBy.distinct) {
            params.set('pagination.sortOption.aggregateBy.distinct', 'true');
        }
    }
}

function listURL(path: string, { query, page, perPage, sortOption }: FetchListParams): string {
    const params = new URLSearchParams();
    if (query) {
        params.set('query', query);
    }
    appendPaginationAndSort(params, { page, perPage, sortOption });
    return `${path}?${params.toString()}`;
}

export function fetchCVEs(params: FetchListParams): Promise<CVEListResponse> {
    return getCached<CVEListResponse>(listURL('/api/v2/vulnmgmt/cves', params));
}

export function fetchImages(params: FetchListParams): Promise<ImageListResponse> {
    return getCached<ImageListResponse>(listURL('/api/v2/vulnmgmt/images', params));
}

export function fetchDeployments(params: FetchListParams): Promise<DeploymentListResponse> {
    return getCached<DeploymentListResponse>(listURL('/api/v2/vulnmgmt/deployments', params));
}

export function fetchCVEDetail(cve: string, query?: string): Promise<CVEDetailResponse> {
    const params = new URLSearchParams();
    if (query) {
        params.set('query', query);
    }
    const suffix = params.toString() ? `?${params.toString()}` : '';
    return getCached<CVEDetailResponse>(
        `/api/v2/vulnmgmt/cves/${encodeURIComponent(cve)}/detail${suffix}`
    );
}

export function fetchVulnMgmtCounts(query: string): Promise<VulnMgmtCounts> {
    const params = new URLSearchParams();
    if (query) {
        params.set('query', query);
    }
    return getCached<VulnMgmtCounts>(`/api/v2/vulnmgmt/counts?${params.toString()}`);
}
