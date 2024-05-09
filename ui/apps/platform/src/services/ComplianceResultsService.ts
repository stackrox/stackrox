import axios from 'services/instance';

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
export type ComplianceCheckStatusCount = {
    count: number;
    status: ComplianceCheckStatus;
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

/**
 * Fetches the scan stats grouped by profile.
 */
export function getComplianceProfilesStats(): Promise<ListComplianceProfileScanStatsResponse> {
    return axios
        .get<ListComplianceProfileScanStatsResponse>(`${complianceResultsBaseUrl}/stats/profiles`)
        .then((response) => response.data);
}
