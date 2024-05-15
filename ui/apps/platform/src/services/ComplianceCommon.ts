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
