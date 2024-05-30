import qs from 'qs';
import { generatePath } from 'react-router-dom';

import axios from 'services/instance';
import { ApiSortOption } from 'types/search';
import { getPaginationParams } from 'utils/searchUtils';

import {
    ComplianceCheckResultStatusCount,
    ComplianceCheckStatusCount,
    complianceV2Url,
    ListComplianceClusterOverallStatsResponse,
    ListComplianceProfileResults,
} from './ComplianceCommon';

const complianceResultsStatsBaseUrl = `${complianceV2Url}/scan/stats`;

type ComplianceProfileScanStats = {
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
export function getComplianceProfilesStats(): Promise<ListComplianceProfileScanStatsResponse> {
    return axios
        .get<ListComplianceProfileScanStatsResponse>(`${complianceResultsStatsBaseUrl}/profiles`)
        .then((response) => response.data);
}

/**
 * Fetches the profile cluster results.
 */
export function getComplianceClusterStats(
    profileName: string,
    sortOption: ApiSortOption,
    page: number,
    perPage: number
): Promise<ListComplianceClusterOverallStatsResponse> {
    const queryParameters = {
        query: {
            pagination: getPaginationParams({ page, perPage, sortOption }),
        },
    };
    const params = qs.stringify(queryParameters, { arrayFormat: 'repeat', allowDots: true });
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
