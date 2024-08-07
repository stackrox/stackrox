import qs from 'qs';

import { SearchQueryOptions } from 'types/search';
import { getPaginationParams, getRequestQueryStringForSearchFilter } from 'utils/searchUtils';

export const complianceV2Url = '/v2/compliance';

export const ComplianceCheckStatusValues = [
    'UNSET_CHECK_STATUS',
    'PASS',
    'FAIL',
    'ERROR',
    'INFO',
    'MANUAL',
    'NOT_APPLICABLE',
    'INCONSISTENT',
] as const;

export type ComplianceCheckStatus = (typeof ComplianceCheckStatusValues)[number];

export type ComplianceScanCluster = {
    clusterId: string;
    clusterName: string;
};

export type ComplianceCheckStatusCount = {
    count: number;
    status: ComplianceCheckStatus;
};

export type ComplianceControl = {
    standard: string;
    control: string;
};

export type ComplianceCheckResultStatusCount = {
    checkName: string;
    rationale: string;
    ruleName: string;
    checkStats: ComplianceCheckStatusCount[];
    controls: ComplianceControl[];
};

export type ListComplianceProfileResults = {
    profileResults: ComplianceCheckResultStatusCount[];
    profileName: string;
    totalCount: number;
};

export type ComplianceClusterOverallStats = {
    cluster: ComplianceScanCluster;
    checkStats: ComplianceCheckStatusCount[];
    clusterErrors: string[];
    lastScanTime: string; // ISO 8601 date string
};

export type ListComplianceClusterOverallStatsResponse = {
    scanStats: ComplianceClusterOverallStats[];
    totalCount: number;
};

export type ComplianceBenchmark = {
    name: string;
    version: string;
    description: string;
    provider: string;
    // shortName is extracted from the annotation.
    // Example: from https://control.compliance.openshift.io/CIS-OCP we should have CIS-OCP
    shortName: string;
};

export type ComplianceProfileSummary = {
    name: string;
    productType: string;
    description: string;
    title: string;
    ruleCount: number;
    profileVersion: string;
    standards: ComplianceBenchmark[];
};

/*
 * Builds query parameters for nested RawQuery in compliance API requests
 *
 * This is used when the RawQuery is nested within the request parameter,
 * not when it's the sole parameter.
 */
export function buildNestedRawQueryParams({
    page,
    perPage,
    sortOption,
    searchFilter = {},
}: SearchQueryOptions): string {
    const query = getRequestQueryStringForSearchFilter(searchFilter);
    const pagination = getPaginationParams({ page, perPage, sortOption });
    const queryParameters = {
        query: {
            query,
            pagination,
        },
    };
    return qs.stringify(queryParameters, { arrayFormat: 'repeat', allowDots: true });
}
