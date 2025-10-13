package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ComplianceOperatorRuleV2 --references=storage.Cluster --search-category COMPLIANCE_RULES --feature-flag ComplianceEnhancements
