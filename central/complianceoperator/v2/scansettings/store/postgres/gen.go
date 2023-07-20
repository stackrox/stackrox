package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ComplianceOperatorScanSettingV2 --search-category COMPLIANCE_SCAN_SETTINGS --references=storage.Cluster,storage.ComplianceOperatorProfileV2 --get-all-func
