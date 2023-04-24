// Semicolon on separate line following the strings prevents an extra changed line to add a string at the end.
// However, add strings in alphabetical order to minimize merge conflicts when multiple people add strings.
// prettier-ignore
export type FeatureFlagEnvVar =
    | 'ROX_ACTIVE_VULN_MGMT'
    | 'ROX_CLAIR_V4_SCANNING'
    | 'ROX_DECLARATIVE_CONFIGURATION'
    | 'ROX_NETPOL_FIELDS'
    | 'ROX_NETWORK_GRAPH_PATTERNFLY'
    | 'ROX_POSTGRES_DATASTORE'
    | 'ROX_SYSLOG_EXTRA_FIELDS'
    | 'ROX_SYSTEM_HEALTH_PF'
    | 'ROX_VULN_MGMT_REPORTING_ENHANCEMENTS'
    | 'ROX_VULN_MGMT_WORKLOAD_CVES'
    ;
