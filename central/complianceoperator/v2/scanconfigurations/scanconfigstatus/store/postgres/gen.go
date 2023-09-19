package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ComplianceOperatorClusterScanConfigStatus --search-category COMPLIANCE_SCAN_CONFIG_STATUS --references=storage.Cluster,storage.ComplianceOperatorScanConfigurationV2 --get-all-func --feature-flag ComplianceEnhancements
