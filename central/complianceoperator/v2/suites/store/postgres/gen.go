package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ComplianceOperatorSuite --feature-flag ComplianceEnhancements --references storage.Cluster --search-category COMPLIANCE_SUITES
