package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ComplianceOperatorProfileV2 --references=storage.ComplianceOperatorRuleV2 --search-category COMPLIANCE_PROFILES --feature-flag ComplianceEnhancements
