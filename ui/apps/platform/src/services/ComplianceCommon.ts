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
    dataState?: ComplianceDataState;
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
    dataState?: ComplianceDataState;
};

export type ListComplianceClusterOverallStatsResponse = {
    scanStats: ComplianceClusterOverallStats[];
    totalCount: number;
    outdatedClusterCount?: number;
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

export const complianceProfileOperatorKindValues = [
    'OPERATOR_KIND_UNSPECIFIED',
    'PROFILE',
    'TAILORED_PROFILE',
] as const;

export type ComplianceProfileOperatorKind = (typeof complianceProfileOperatorKindValues)[number];

export type ComplianceDataState =
    | 'COMPLIANCE_DATA_STATE_UNKNOWN'
    | 'COMPLIANCE_DATA_STATE_CURRENT'
    | 'COMPLIANCE_DATA_STATE_OUTDATED';

export type ComplianceProfileSummary = {
    name: string;
    productType: string;
    description: string;
    title: string;
    ruleCount: number;
    profileVersion: string;
    standards: ComplianceBenchmark[];
    operatorKind?: ComplianceProfileOperatorKind;
};
