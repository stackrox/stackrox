// Semicolon on separate line following the strings prevents an extra changed line to add a string at the end.
// However, add strings in alphabetical order to minimize merge conflicts when multiple people add strings.
// prettier-ignore
export type FeatureFlagEnvVar =
    | 'ROX_ACTIVE_VULN_MGMT'
    | 'ROX_ADMISSION_CONTROLLER_CONFIG'
    | 'ROX_AUTO_LOCK_PROCESS_BASELINES'
    | 'ROX_CLUSTERS_PAGE_MIGRATION_UI'
    | 'ROX_CUSTOMIZABLE_PLATFORM_COMPONENTS'
    | 'ROX_EXTERNAL_IPS'
    | 'ROX_FLATTEN_CVE_DATA'
    | 'ROX_FLATTEN_IMAGE_DATA'
    // | 'ROX_KEV_EXPLOIT' // Ross CISA KEV
    | 'ROX_NETWORK_GRAPH_EXTERNAL_IPS'
    | 'ROX_NODE_INDEX_ENABLED'
    | 'ROX_POLICY_CRITERIA_MODAL'
    | 'ROX_SCANNER_V4'
    | 'ROX_VIRTUAL_MACHINES'
    | 'ROX_VULN_MGMT_LEGACY_SNOOZE'
    | 'ROX_VULNERABILITY_VIEW_BASED_REPORTS'
    ;
