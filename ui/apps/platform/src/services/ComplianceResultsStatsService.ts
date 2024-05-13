import { generatePath } from 'react-router-dom';

import axios from 'services/instance';

import {
    ComplianceCheckResultStatusCount,
    ComplianceCheckStatusCount,
    ListComplianceClusterOverallStatsResponse,
    ListComplianceProfileResults,
    complianceV2Url,
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
    profileName: string
): Promise<ListComplianceClusterOverallStatsResponse> {
    return axios
        .get<ListComplianceClusterOverallStatsResponse>(
            `${complianceResultsStatsBaseUrl}/profiles/${profileName}/clusters`
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
