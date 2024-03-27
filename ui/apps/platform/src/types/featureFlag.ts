// Semicolon on separate line following the strings prevents an extra changed line to add a string at the end.
// However, add strings in alphabetical order to minimize merge conflicts when multiple people add strings.
// prettier-ignore
export type FeatureFlagEnvVar =
    | 'ROX_ACTIVE_VULN_MGMT'
    | 'ROX_ADMINISTRATION_EVENTS'
    | 'ROX_CLOUD_CREDENTIALS'
    | 'ROX_CLOUD_SOURCES'
    | 'ROX_COMPLIANCE_ENHANCEMENTS'
    | 'ROX_COMPLIANCE_HIERARCHY_CONTROL_DATA'
    | 'ROX_COMPLIANCE_REPORTING'
    | 'ROX_MOVE_INIT_BUNDLES_UI'
    | 'ROX_POLICY_CRITERIA_MODAL'
    | 'ROX_VULN_MGMT_REPORTING_ENHANCEMENTS'
    | 'ROX_VULN_MGMT_UNIFIED_CVE_DEFERRAL'
    | 'ROX_VULN_MGMT_WORKLOAD_CVES'
    | 'ROX_VULN_MGMT_NODE_PLATFORM_CVES'
    | 'ROX_WORKLOAD_CVES_FIXABILITY_FILTERS'
    | 'ROX_SCANNER_V4'
    ;
