// Semicolon on separate line following the strings prevents an extra changed line to add a string at the end.
// However, add strings in alphabetical order to minimize merge conflicts when multiple people add strings.
// prettier-ignore
export type FeatureFlagEnvVar =
    | 'ROX_ACSCS_EMAIL_NOTIFIER'
    | 'ROX_ACTIVE_VULN_MGMT'
    | 'ROX_CLUSTERS_PAGE_MIGRATION_UI'
    | 'ROX_COMPLIANCE_ENHANCEMENTS'
    | 'ROX_COMPLIANCE_HIERARCHY_CONTROL_DATA'
    | 'ROX_COMPLIANCE_REPORTING'
    | 'ROX_NVD_CVSS_UI'
    | 'ROX_PLATFORM_COMPONENTS'
    | 'ROX_POLICY_CRITERIA_MODAL'
    | 'ROX_SCAN_SCHEDULE_REPORT_JOBS'
    | 'ROX_SCANNER_V4'
    | 'ROX_VULN_MGMT_2_GA'
    | 'ROX_VULN_MGMT_ADVANCED_FILTERS'
    | 'ROX_VULN_MGMT_LEGACY_SNOOZE'
    | 'ROX_VULN_MGMT_NODE_PLATFORM_CVES'
    | 'ROX_VULN_MGMT_REPORTING_ENHANCEMENTS'
    | 'ROX_VULN_MGMT_UNIFIED_CVE_DEFERRAL'
    | 'ROX_VULN_MGMT_WORKLOAD_CVES'
    | 'ROX_WORKLOAD_CVES_FIXABILITY_FILTERS'
    ;
