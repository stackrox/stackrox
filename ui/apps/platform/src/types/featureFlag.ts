// Semicolon on separate line following the strings prevents an extra changed line to add a string at the end.
// However, add strings in alphabetical order to minimize merge conflicts when multiple people add or delete strings.
// prettier-ignore
export type FeatureFlagEnvVar =
    | 'ROX_BASE_IMAGE_DETECTION'
    | 'ROX_CISA_KEV'
    | 'ROX_CVE_FIX_TIMESTAMP'
    | 'ROX_FLATTEN_IMAGE_DATA'
    | 'ROX_NODE_INDEX_ENABLED'
    | 'ROX_POLICY_CRITERIA_MODAL'
    | 'ROX_SCANNER_V4'
    | 'ROX_SENSITIVE_FILE_ACTIVITY'
    | 'ROX_VIRTUAL_MACHINES'
    | 'ROX_VULN_MGMT_LEGACY_SNOOZE'
    ;
