import qs from 'qs';

import axios from 'services/instance';
import { ApiSortOption } from 'types/search';
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
    lastScanTime: string; // ISO 8601 date string
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

export type ListComplianceCheckClusterResponse = {
    checkResults: ClusterCheckStatus[];
    profileName: string;
    checkName: string;
    totalCount: number;
};

/**
 * Fetches statuses per cluster based off a single check.
 */
export function getComplianceProfileCheckResult(
    profileName: string,
    checkName: string,
    page: number,
    perPage: number
): Promise<ListComplianceCheckClusterResponse> {
    const queryParameters = {
        query: {
            pagination: getPaginationParams({ page, perPage }),
        },
    };
    const params = qs.stringify(queryParameters, { arrayFormat: 'repeat', allowDots: true });
    return axios
        .get<ListComplianceCheckClusterResponse>(
            `${complianceResultsBaseUrl}/results/profiles/${profileName}/checks/${checkName}?${params}`
        )
        .then((response) => response.data);
}

/**
 * Fetches the profile check results.
 */
export function getComplianceProfileResults(
    profileName: string,
    sortOption: ApiSortOption,
    page: number,
    perPage: number
): Promise<ListComplianceProfileResults> {
    const queryParameters = {
        query: {
            pagination: getPaginationParams({ page, perPage, sortOption }),
        },
    };
    const params = qs.stringify(queryParameters, { arrayFormat: 'repeat', allowDots: true });

    return axios
        .get<ListComplianceProfileResults>(
            `${complianceResultsBaseUrl}/results/profiles/${profileName}/checks?${params}`
        )
        .then((response) => response.data);
}
