import { generatePath } from 'react-router-dom';

import axios from 'services/instance';

import { ComplianceCheckStatusCount, complianceV2Url } from './ComplianceCommon';

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
        .get<ListComplianceProfileScanStatsResponse>(`${complianceV2Url}/scan/stats/profiles`)
        .then((response) => response.data);
}

export type ComplianceCheckResultStatusCount = {
    checkName: string;
    rationale: string;
    ruleName: string;
    checkStats: ComplianceCheckStatusCount[];
};

export type ListComplianceProfileResults = {
    profileResults: ComplianceCheckResultStatusCount[];
    profileName: string;
    totalCount: number;
};

/*
 * Fetches the scan stats for a specific profile check
 */
export function getComplianceProfileCheckStats(
    profileName: string,
    checkName: string
): Promise<ComplianceCheckResultStatusCount> {
    const url = generatePath(
        `${complianceV2Url}/scan/stats/profile/:profileName/checks/:checkName`,
        { profileName, checkName }
    );
    return axios
        .get<ListComplianceProfileResults>(url)
        .then((response) => response.data?.profileResults?.[0]);
}
