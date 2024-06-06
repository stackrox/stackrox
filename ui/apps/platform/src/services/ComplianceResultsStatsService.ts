import { generatePath } from 'react-router-dom';
import qs from 'qs';

import axios from 'services/instance';
import { SearchQueryOptions } from 'types/search';
import { getRequestQueryStringForSearchFilter } from 'utils/searchUtils';

import {
    buildNestedRawQueryParams,
    ComplianceCheckResultStatusCount,
    ComplianceCheckStatusCount,
    complianceV2Url,
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

/**
 * Fetches the scan stats grouped by profile.
 */
export function getComplianceProfilesStats(
    clusterId: string = ''
): Promise<ListComplianceProfileScanStatsResponse> {
    let params = '';
    if (clusterId) {
        const searchQuery = getRequestQueryStringForSearchFilter({
            'Cluster ID': clusterId,
        });

        params = qs.stringify({ query: searchQuery }, { arrayFormat: 'repeat', allowDots: true });
    }

    return axios
        .get<ListComplianceProfileScanStatsResponse>(
            `${complianceResultsStatsBaseUrl}/profiles?${params}`
        )
        .then((response) => response.data);
}

/**
 * Fetches the profile cluster results.
 */
export function getComplianceClusterStats(
    profileName: string,
    { sortOption, page, perPage }: SearchQueryOptions
): Promise<ListComplianceClusterOverallStatsResponse> {
    const params = buildNestedRawQueryParams({ page, perPage, sortOption });
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
    checkName: string
): Promise<ComplianceCheckResultStatusCount> {
    const url = generatePath(
        `${complianceV2Url}/scan/stats/profiles/:profileName/checks/:checkName`,
        { profileName, checkName }
    );
    return axios
        .get<ListComplianceProfileResults>(url)
        .then((response) => response.data?.profileResults?.[0]);
}
