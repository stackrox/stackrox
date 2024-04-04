package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ComplianceOperatorControlV2 --references=storage.ComplianceOperatorBenchmarkV2 --search-category COMPLIANCE_BENCHMARK_CONTROL
