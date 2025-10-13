package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ComplianceOperatorScanSettingBindingV2 --references=storage.Cluster --search-category COMPLIANCE_SCAN_SETTING_BINDINGS --feature-flag ComplianceEnhancements
