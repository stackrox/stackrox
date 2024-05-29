import qs from 'qs';

import { SearchQueryOptions } from 'types/search';
import { getPaginationParams } from 'utils/searchUtils';

export const complianceV2Url = '/v2/compliance';

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

export type ComplianceScanCluster = {
    clusterId: string;
    clusterName: string;
};

export type ComplianceCheckStatusCount = {
    count: number;
    status: ComplianceCheckStatus;
};

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
}: SearchQueryOptions): string {
    const queryParameters = {
        query: {
            pagination: getPaginationParams({ page, perPage, sortOption }),
        },
    };
    return qs.stringify(queryParameters, { arrayFormat: 'repeat', allowDots: true });
}
