package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ComplianceOperatorReportSnapshotV2 --search-category COMPLIANCE_REPORT_SNAPSHOT --feature-flag ComplianceEnhancements --references storage.ComplianceOperatorScanConfigurationV2,storage.ComplianceOperatorScanV2
