import axios from 'services/instance';
import qs from 'qs';

import { SearchFilter, ApiSortOption } from 'types/search';
import { getListQueryParams, getRequestQueryStringForSearchFilter } from 'utils/searchUtils';

import { complianceV2Url } from './ComplianceCommon';

const complianceResultsBaseUrl = `${complianceV2Url}/scan`;

export const ComplianceCheckStatusEnum = {
    UNSET_CHECK_STATUS: 'UNSET_CHECK_STATUS',
    PASS: 'PASS',
    FAIL: 'FAIL',
    ERROR: 'ERROR',
    INFO: 'INFO',
    MANUAL: 'MANUAL',
    NOT_APPLICABLE: 'NOT_APPLICABLE',
    INCONSISTENT: 'INCONSISTENT',
} as const;

export type ComplianceCheckStatus =
    (typeof ComplianceCheckStatusEnum)[keyof typeof ComplianceCheckStatusEnum];

type ComplianceScanCluster = {
    clusterId: string;
    clusterName: string;
};

export type ClusterCheckStatus = {
    cluster: ComplianceScanCluster;
    status: ComplianceCheckStatus;
    createdTime: string; // ISO 8601 date string
    checkUid: string;
};

export type ComplianceCheckResult = {
    checkId: string;
    checkName: string;
    clusters: ClusterCheckStatus[];
    description: string;
    instructions: string;
    standard: string;
    control: string;
    rationale: string;
    valuesUsed: string[];
    warnings: string[];
};

type ComplianceScanResult = {
    scanName: string;
    profileName: string;
    checkResults: ComplianceCheckResult[];
    scanConfigId: string;
};

export type ComplianceCheckStatusCount = {
    count: number;
    status: ComplianceCheckStatus;
};

export type ComplianceScanStatsShim = {
    scanName: string;
    checkStats: ComplianceCheckStatusCount[];
    lastScan: string; // ISO 8601 date string
    scanConfigId: string;
};

export type ComplianceClusterScanStats = {
    scanStats: ComplianceScanStatsShim;
    cluster: ComplianceScanCluster;
};

export type ComplianceClusterOverallStats = {
    cluster: ComplianceScanCluster;
    checkStats: ComplianceCheckStatusCount[];
    clusterErrors: string[];
};

/**
 * Fetches stats for all clusters
 * Note: this function and getSingleClusterCombinedStats call the same API endpoint
 * due to the absence of a dedicated single-cluster endpoint
 */
export function getAllClustersCombinedStats(
    sortOption: ApiSortOption,
    page?: number,
    pageSize?: number
): Promise<ComplianceClusterOverallStats[]> {
    const searchFilter = {};
    const params = getListQueryParams(searchFilter, sortOption, page, pageSize);

    return axios
        .get<{
            scanStats: ComplianceClusterOverallStats[];
        }>(`${complianceResultsBaseUrl}/stats/overall/cluster?${params}`)
        .then((response) => {
            return response?.data?.scanStats ?? [];
        });
}

/**
 * Fetches the count of clusters stats.
 */
export function getAllClustersCombinedStatsCount(): Promise<number> {
    return axios
        .get<{
            count: number;
        }>(`${complianceResultsBaseUrl}/stats/overall/cluster/count`)
        .then((response) => {
            return response?.data?.count ?? 0;
        });
}

/**
 * Fetches stats for a single cluster
 * Note: this function and getAllClustersCombinedStats call the same API endpoint
 * due to the absence of a dedicated single-cluster endpoint
 */
export function getSingleClusterCombinedStats(
    clusterId: string
): Promise<ComplianceClusterOverallStats | null> {
    const query = getRequestQueryStringForSearchFilter({
        'Cluster ID': clusterId,
    });
    const params = qs.stringify({ query });

    return axios
        .get<{
            scanStats: ComplianceClusterOverallStats[];
        }>(`${complianceResultsBaseUrl}/stats/overall/cluster?${params}`)
        .then((response) => {
            const stats = response?.data?.scanStats;
            return stats && stats.length > 0 ? stats[0] : null;
        });
}

/**
 * Fetches stats for a single cluster grouped by scan config
 * Note: this function and getAllClustersStatsByScanConfig call the same API endpoint
 * due to the absence of a dedicated single-cluster endpoint
 */
export function getSingleClusterStatsByScanConfig(
    clusterId: string
): Promise<ComplianceClusterScanStats[] | null> {
    const query = getRequestQueryStringForSearchFilter({
        'Cluster ID': clusterId,
    });
    const params = qs.stringify({ query });

    return axios
        .get<{
            scanStats: ComplianceClusterScanStats[];
        }>(`${complianceResultsBaseUrl}/stats/cluster?${params}`)
        .then((response) => {
            return response?.data?.scanStats ?? [];
        });
}

export function getSingleClusterResultsByScanConfig(
    clusterId: string,
    scanName: string,
    searchFilter: SearchFilter,
    sortOption: ApiSortOption,
    page?: number,
    pageSize?: number
): Promise<ComplianceScanResult | null> {
    if (!clusterId || !scanName) {
        return Promise.reject(new Error('clusterId and scanName are required'));
    }
    const searchQuery = getRequestQueryStringForSearchFilter({
        'Cluster ID': clusterId,
        ...searchFilter,
    });

    let offset: number | undefined;
    if (typeof page === 'number' && typeof pageSize === 'number') {
        offset = page > 0 ? page * pageSize : 0;
    }
    const queryParameters = {
        query: {
            query: searchQuery,
            pagination: { offset, limit: pageSize, sortOption },
        },
    };
    const params = qs.stringify(queryParameters, { arrayFormat: 'repeat', allowDots: true });

    return axios
        .get<{
            scanResults: ComplianceScanResult[];
        }>(`${complianceResultsBaseUrl}/results/${scanName}?${params}`)
        .then((response) => {
            const results = response?.data?.scanResults ?? [];
            if (results.length > 1) {
                throw new Error('Expected a single result set, but received multiple');
            }
            return results[0] || null;
        });
}

export function getSingleClusterResultsByScanConfigCount(
    clusterId: string,
    scanName: string,
    searchFilter: SearchFilter
): Promise<number> {
    const searchQuery = getRequestQueryStringForSearchFilter({
        'Cluster ID': clusterId,
        ...searchFilter,
    });

    const queryParameters = {
        query: {
            query: searchQuery,
        },
    };
    const params = qs.stringify(queryParameters, { arrayFormat: 'repeat', allowDots: true });

    return axios
        .get<{
            count: number;
        }>(`${complianceResultsBaseUrl}/count/results/${scanName}?${params}`)
        .then((response) => {
            return response?.data?.count ?? 0;
        });
}
