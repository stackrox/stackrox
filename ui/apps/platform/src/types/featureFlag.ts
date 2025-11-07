// Semicolon on separate line following the strings prevents an extra changed line to add a string at the end.
// However, add strings in alphabetical order to minimize merge conflicts when multiple people add strings.
// prettier-ignore
export type FeatureFlagEnvVar =
    | 'ROX_CISA_KEV'
    | 'ROX_EXTERNAL_IPS'
    | 'ROX_FLATTEN_IMAGE_DATA'
    | 'ROX_NETWORK_GRAPH_EXTERNAL_IPS'
    | 'ROX_NODE_INDEX_ENABLED'
    | 'ROX_POLICY_CRITERIA_MODAL'
    | 'ROX_SCANNER_V4'
    | 'ROX_VIRTUAL_MACHINES'
    | 'ROX_VULN_MGMT_LEGACY_SNOOZE'
    | 'ROX_VULNERABILITY_VIEW_BASED_REPORTS'
    ;
