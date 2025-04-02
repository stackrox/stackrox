// Semicolon on separate line following the strings prevents an extra changed line to add a string at the end.
// However, add strings in alphabetical order to minimize merge conflicts when multiple people add strings.
// prettier-ignore
export type FeatureFlagEnvVar =
    | 'ROX_ACTIVE_VULN_MGMT'
    | 'ROX_CLUSTERS_PAGE_MIGRATION_UI'
    | 'ROX_CUSTOMIZABLE_PLATFORM_COMPONENTS'
    | 'ROX_CVE_ADVISORY_SEPARATION'
    | 'ROX_EXTERNAL_IPS'
    | 'ROX_NETWORK_GRAPH_EXTERNAL_IPS'
    | 'ROX_NODE_INDEX_ENABLED'
    | 'ROX_PLATFORM_CVE_SPLIT'
    | 'ROX_POLICY_CRITERIA_MODAL'
    | 'ROX_SBOM_GENERATION'
    | 'ROX_SCAN_SCHEDULE_REPORT_JOBS'
    | 'ROX_SCANNER_V4'
    | 'ROX_VULN_MGMT_LEGACY_SNOOZE'
    | 'ROX_VULNERABILITY_ON_DEMAND_REPORTS'
    ;
