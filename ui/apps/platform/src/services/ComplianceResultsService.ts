import axios from 'services/instance';

import {
    ComplianceCheckStatusCount,
    ComplianceCheckStatus,
    complianceV2Url,
} from './ComplianceCommon';

const complianceResultsBaseUrl = `${complianceV2Url}/scan`;

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

export type ComplianceCheckResultStatusCount = {
    checkName: string;
    rationale: string;
    ruleName: string;
    checkStats: ComplianceCheckStatusCount[];
};

type ListComplianceProfileResults = {
    profileResults: ComplianceCheckResultStatusCount[];
    profileName: string;
    totalCount: number;
};

/**
 * Fetches the scan stats grouped by profile.
 */
export function getComplianceProfilesStats(): Promise<ListComplianceProfileScanStatsResponse> {
    return axios
        .get<ListComplianceProfileScanStatsResponse>(`${complianceResultsBaseUrl}/stats/profiles`)
        .then((response) => response.data);
}

/**
 * Fetches the profile check results.
 */
export function getComplianceProfileResults(
    profileName: string
): Promise<ListComplianceProfileResults> {
    return axios
        .get<ListComplianceProfileResults>(
            `${complianceResultsBaseUrl}/results/profiles/${profileName}/checks`
        )
        .then((response) => response.data);
}
