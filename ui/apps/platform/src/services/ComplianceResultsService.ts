import qs from 'qs';

import axios from 'services/instance';
import { getPaginationParams } from 'utils/searchUtils';

import {
    ComplianceCheckStatus,
    ComplianceScanCluster,
    complianceV2Url,
    ListComplianceProfileResults,
} from './ComplianceCommon';

const complianceResultsBaseUrl = `${complianceV2Url}/scan`;

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
    profileName: string,
    page: number,
    perPage: number
): Promise<ListComplianceProfileResults> {
    const queryParameters = {
        query: {
            pagination: getPaginationParams(page, perPage),
        },
    };
    const params = qs.stringify(queryParameters, { arrayFormat: 'repeat', allowDots: true });

    return axios
        .get<ListComplianceProfileResults>(
            `${complianceResultsBaseUrl}/results/profiles/${profileName}/checks?${params}`
        )
        .then((response) => response.data);
}
