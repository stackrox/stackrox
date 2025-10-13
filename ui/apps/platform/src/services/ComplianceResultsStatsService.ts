import axios from 'services/instance';
import type { SearchFilter, SearchQueryOptions } from 'types/search';
import qs from 'qs';

import { getRequestQueryStringForSearchFilter } from 'utils/searchUtils';

import { buildNestedRawQueryParams, complianceV2Url } from './ComplianceCommon';
import type {
    ComplianceCheckResultStatusCount,
    ComplianceCheckStatusCount,
    ListComplianceClusterOverallStatsResponse,
    ListComplianceProfileResults,
} from './ComplianceCommon';

const complianceResultsStatsBaseUrl = `${complianceV2Url}/scan/stats`;

export type ComplianceProfileScanStats = {
    checkStats: ComplianceCheckStatusCount[];
    profileName: string;
    title: string;
    version: string;
};

export type ListComplianceProfileScanStatsResponse = {
    scanStats: ComplianceProfileScanStats[];
    totalCount: number;
};

export type ListComplianceClusterProfileStatsResponse = {
    scanStats: ComplianceProfileScanStats[];
    totalCount: number;
    clusterId: string;
    clusterName: string;
};

/**
 * Fetches the scan stats grouped by profile.
 */
export function getComplianceProfilesStats(
    scanConfigSearchFilter: SearchFilter
): Promise<ListComplianceProfileScanStatsResponse> {
    const query = getRequestQueryStringForSearchFilter(scanConfigSearchFilter);
    const params = qs.stringify({ query }, { arrayFormat: 'repeat', allowDots: true });
    return axios
        .get<ListComplianceProfileScanStatsResponse>(
            `${complianceResultsStatsBaseUrl}/profiles?${params}`
        )
        .then((response) => response.data);
}

/**
 * Fetches the scan stats grouped by profile for a specific cluster.
 */
export function getComplianceProfilesClusterStats(
    clusterId: string
): Promise<ListComplianceClusterProfileStatsResponse> {
    return axios
        .get<ListComplianceClusterProfileStatsResponse>(
            `${complianceResultsStatsBaseUrl}/profiles/clusters/${clusterId}`
        )
        .then((response) => response.data);
}

/**
 * Fetches the profile cluster results.
 */
export function getComplianceClusterStats(
    profileName: string,
    { sortOption, page, perPage, searchFilter }: SearchQueryOptions
): Promise<ListComplianceClusterOverallStatsResponse> {
    const params = buildNestedRawQueryParams({ page, perPage, sortOption, searchFilter });
    return axios
        .get<ListComplianceClusterOverallStatsResponse>(
            `${complianceResultsStatsBaseUrl}/profiles/${profileName}/clusters?${params}`
        )
        .then((response) => response.data);
}

/*
 * Fetches the scan stats for a specific profile check
 */
export function getComplianceProfileCheckStats(
    profileName: string,
    checkName: string,
    scanConfigSearchFilter: SearchFilter
): Promise<ComplianceCheckResultStatusCount> {
    const query = getRequestQueryStringForSearchFilter(scanConfigSearchFilter);
    const params = qs.stringify({ query: { query } }, { arrayFormat: 'repeat', allowDots: true });
    return axios
        .get<ListComplianceProfileResults>(
            `${complianceResultsStatsBaseUrl}/profiles/${profileName}/checks/${checkName}?${params}`
        )
        .then((response) => response.data?.profileResults?.[0]);
}
