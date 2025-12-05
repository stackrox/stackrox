package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ComplianceOperatorProfileV2 --references=storage.ComplianceOperatorRuleV2 --search-category COMPLIANCE_PROFILES --search-scope COMPLIANCE_PROFILES,COMPLIANCE_SCAN_CONFIG --feature-flag ComplianceEnhancements
