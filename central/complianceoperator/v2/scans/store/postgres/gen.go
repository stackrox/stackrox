package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ComplianceOperatorScanV2 --search-category COMPLIANCE_SCAN --references=storage.Cluster,storage.ComplianceOperatorProfileV2,storage.ComplianceOperatorScanConfigurationV2 --feature-flag ComplianceEnhancements
