import axios from 'services/instance';
import type { SearchQueryOptions } from 'types/search';

import { buildNestedRawQueryParams, complianceV2Url } from './ComplianceCommon';
import type {
    ComplianceCheckStatus,
    ComplianceControl,
    ComplianceScanCluster,
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
    controls: ComplianceControl[];
    rationale: string;
    valuesUsed: string[];
    warnings: string[];
    status: ComplianceCheckStatus;
    ruleName: string;
    labels: { [key: string]: string };
    annotations: { [key: string]: string };
};

export type ComplianceClusterCheckStatus = {
    checkId: string;
    checkName: string;
    clusters: ClusterCheckStatus[];
    description: string;
    instructions: string;
    controls: ComplianceControl[];
    rationale: string;
    valuesUsed: string[];
    warnings: string[];
    labels: { [key: string]: string };
    annotations: { [key: string]: string };
};

export type ListComplianceCheckClusterResponse = {
    checkResults: ClusterCheckStatus[];
    profileName: string;
    checkName: string;
    totalCount: number;
    controls: ComplianceControl[];
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
    { sortOption, page, perPage, searchFilter }: SearchQueryOptions
): Promise<ListComplianceCheckClusterResponse> {
    const params = buildNestedRawQueryParams({ page, perPage, sortOption, searchFilter });
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
    { sortOption, page, perPage, searchFilter }: SearchQueryOptions
): Promise<ListComplianceProfileResults> {
    const params = buildNestedRawQueryParams({ page, perPage, sortOption, searchFilter });
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
    clusterId: string,
    { sortOption, page, perPage, searchFilter }: SearchQueryOptions
): Promise<ListComplianceCheckResultResponse> {
    const params = buildNestedRawQueryParams({ page, perPage, sortOption, searchFilter });
    return axios
        .get<ListComplianceCheckResultResponse>(
            `${complianceResultsBaseUrl}/results/profiles/${profileName}/clusters/${clusterId}?${params}`
        )
        .then((response) => response.data);
}

/**
 * Fetches check details.
 */
export function getComplianceProfileCheckDetails(
    profileName: string,
    checkName: string
): Promise<ComplianceClusterCheckStatus> {
    return axios
        .get<ComplianceClusterCheckStatus>(
            `${complianceResultsBaseUrl}/results/profiles/${profileName}/checks/${checkName}/details`
        )
        .then((response) => response.data);
}
