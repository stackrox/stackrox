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
    dataFreshness: ComplianceAggregateDataFreshness;
};

export type ListComplianceProfileResults = {
    profileResults: ComplianceCheckResultStatusCount[];
    profileName: string;
    totalCount: number;
};

// ComplianceCheckDataFreshness indicates whether a single compliance check
// result is current relative to the scan configuration's schedule.
export type ComplianceCheckDataFreshness =
    // No freshness information available. Either the server could not evaluate
    // freshness (no schedule, unresolvable config, or no data) or the field was
    // not populated. This is the zero value.
    | 'COMPLIANCE_CHECK_DATA_FRESHNESS_UNKNOWN'
    // Data reflects the most recent expected scan cycle.
    | 'COMPLIANCE_CHECK_DATA_FRESHNESS_CURRENT'
    // Data predates the most recent expected scan cycle.
    | 'COMPLIANCE_CHECK_DATA_FRESHNESS_STALE';

// ComplianceAggregateDataFreshness indicates the freshness of compliance data
// aggregated over multiple checks (a cluster, a scan, or a cross-cluster rollup of
// a single check). Unlike a single check, an aggregate can mix fresh and stale
// results.
export type ComplianceAggregateDataFreshness =
    // No freshness information available. Either no contributing check could be
    // resolved to current or stale, or the field was not populated. This is the
    // zero value.
    | 'COMPLIANCE_AGGREGATE_DATA_FRESHNESS_UNKNOWN'
    // At least one contributing check is current and none are stale.
    | 'COMPLIANCE_AGGREGATE_DATA_FRESHNESS_CURRENT'
    // At least one contributing check is stale and none are current.
    | 'COMPLIANCE_AGGREGATE_DATA_FRESHNESS_STALE'
    // Contributing checks are a mix of stale and current data.
    | 'COMPLIANCE_AGGREGATE_DATA_FRESHNESS_INCONSISTENT';

export type ComplianceClusterOverallStats = {
    cluster: ComplianceScanCluster;
    checkStats: ComplianceCheckStatusCount[];
    clusterErrors: string[];
    lastScanTime: string; // ISO 8601 date string
    dataFreshness: ComplianceAggregateDataFreshness;
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

export const complianceProfileOperatorKindValues = [
    'OPERATOR_KIND_UNSPECIFIED',
    'PROFILE',
    'TAILORED_PROFILE',
] as const;

export type ComplianceProfileOperatorKind = (typeof complianceProfileOperatorKindValues)[number];

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
