import axios from 'services/instance';
import { SearchQueryOptions } from 'types/search';

import {
    buildNestedRawQueryParams,
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
    checkUid: string;
    description: string;
    instructions: string;
    standard: string;
    control: string[];
    rationale: string;
    valuesUsed: string[];
    warnings: string[];
    status: ComplianceCheckStatus;
    ruleName: string;
};

export type ListComplianceCheckClusterResponse = {
    checkResults: ClusterCheckStatus[];
    profileName: string;
    checkName: string;
    totalCount: number;
};

export type ListComplianceCheckResultResponse = {
    checkResults: ComplianceCheckResult[];
    profileName: string;
    clusterId: string;
    totalCount: number;
    lastScanTime: string; // ISO 8601 date string
};

/**
 * Fetches statuses per cluster based off a single check.
 */
export function getComplianceProfileCheckResult(
    profileName: string,
    checkName: string,
    { sortOption, page, perPage }: SearchQueryOptions
): Promise<ListComplianceCheckClusterResponse> {
    const params = buildNestedRawQueryParams({ page, perPage, sortOption });
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
    { sortOption, page, perPage }: SearchQueryOptions
): Promise<ListComplianceProfileResults> {
    const params = buildNestedRawQueryParams({ page, perPage, sortOption });
    return axios
        .get<ListComplianceProfileResults>(
            `${complianceResultsBaseUrl}/results/profiles/${profileName}/checks?${params}`
        )
        .then((response) => response.data);
}

/**
 * Fetches check results based off a cluster and profile.
 */
export function getComplianceProfileClusterResults(
    profileName: string,
    clusterId: string
): Promise<ListComplianceCheckResultResponse> {
    return axios
        .get<ListComplianceCheckResultResponse>(
            `${complianceResultsBaseUrl}/results/profiles/${profileName}/clusters/${clusterId}`
        )
        .then((response) => response.data);
}
