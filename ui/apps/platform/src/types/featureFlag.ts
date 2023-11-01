// Semicolon on separate line following the strings prevents an extra changed line to add a string at the end.
// However, add strings in alphabetical order to minimize merge conflicts when multiple people add strings.
// prettier-ignore
export type FeatureFlagEnvVar =
    | 'ROX_ACTIVE_VULN_MGMT'
    | 'ROX_ADMINISTRATION_EVENTS'
    | 'ROX_COMPLIANCE_ENHANCEMENTS'
    | 'ROX_MOVE_INIT_BUNDLES_UI'
    | 'ROX_POLICY_CRITERIA_MODAL'
    | 'ROX_VULN_MGMT_REPORTING_ENHANCEMENTS'
    | 'ROX_VULN_MGMT_UNIFIED_CVE_DEFERRAL'
    | 'ROX_VULN_MGMT_WORKLOAD_CVES'
    ;
