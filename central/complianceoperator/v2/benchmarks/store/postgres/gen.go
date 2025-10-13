package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ComplianceOperatorBenchmarkV2 --search-category COMPLIANCE_BENCHMARKS --feature-flag ComplianceEnhancements --references storage.ComplianceOperatorProfileV2
