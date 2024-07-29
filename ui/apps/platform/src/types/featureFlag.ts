// Semicolon on separate line following the strings prevents an extra changed line to add a string at the end.
// However, add strings in alphabetical order to minimize merge conflicts when multiple people add strings.
// prettier-ignore
export type FeatureFlagEnvVar =
    | 'ROX_ACSCS_EMAIL_NOTIFIER'
    | 'ROX_ACTIVE_VULN_MGMT'
    | 'ROX_COMPLIANCE_ENHANCEMENTS'
    | 'ROX_COMPLIANCE_HIERARCHY_CONTROL_DATA'
    | 'ROX_COMPLIANCE_REPORTING'
    | 'ROX_POLICY_CRITERIA_MODAL'
    | 'ROX_POLICY_VIOLATIONS_ADVANCED_FILTERS'
    | 'ROX_VULN_MGMT_2_GA'
    | 'ROX_VULN_MGMT_ADVANCED_FILTERS'
    | 'ROX_VULN_MGMT_LEGACY_SNOOZE'
    | 'ROX_VULN_MGMT_NODE_PLATFORM_CVES'
    | 'ROX_VULN_MGMT_REPORTING_ENHANCEMENTS'
    | 'ROX_VULN_MGMT_UNIFIED_CVE_DEFERRAL'
    | 'ROX_VULN_MGMT_WORKLOAD_CVES'
    | 'ROX_WORKLOAD_CVES_FIXABILITY_FILTERS'
    | 'ROX_SCANNER_V4'
    ;
