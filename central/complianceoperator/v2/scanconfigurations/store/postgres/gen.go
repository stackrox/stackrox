package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ComplianceOperatorScanConfigurationV2 --search-category COMPLIANCE_SCAN_CONFIG --references=storage.Cluster,storage.ComplianceOperatorProfileV2,storage.Notifier --search-scope COMPLIANCE_SCAN_CONFIG --feature-flag ComplianceEnhancements
