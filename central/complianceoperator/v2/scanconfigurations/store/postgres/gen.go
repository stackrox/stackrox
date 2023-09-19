package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ComplianceOperatorScanConfigurationV2 --search-category COMPLIANCE_SCAN_SETTINGS --references=storage.ComplianceOperatorProfileV2 --get-all-func --feature-flag ComplianceEnhancements
