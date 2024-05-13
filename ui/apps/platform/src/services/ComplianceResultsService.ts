import axios from 'services/instance';

import {
    ComplianceCheckStatus,
    complianceV2Url,
    ListComplianceProfileResults,
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
