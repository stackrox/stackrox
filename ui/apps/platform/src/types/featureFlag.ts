// Semicolon on separate line following the strings prevents an extra changed line to add a string at the end.
// However, add strings in alphabetical order to minimize merge conflicts when multiple people add or delete strings.
export /* prettier-ignore */ type FeatureFlagEnvVar =
    | 'ROX_COMPLIANCE_SURFACE_STALE_DATA'
    | 'ROX_IMAGE_LAYER_FILTER'
    | 'ROX_LIGHTSPEED_RISK_SUMMARY'
    | 'ROX_NODE_INDEX_ENABLED'
    | 'ROX_POLICY_WORKLOAD_KIND_EXCLUSION'
    | 'ROX_SCANNER_V4'
    | 'ROX_SENSITIVE_FILE_ACTIVITY'
    | 'ROX_UI_SECRETS_PAGE_MIGRATION'
    | 'ROX_VIRTUAL_MACHINES'
    | 'ROX_VIRTUAL_MACHINES_ENHANCED_DATA_MODEL'
    | 'ROX_VULN_MGMT_LEGACY_SNOOZE'
    ;
