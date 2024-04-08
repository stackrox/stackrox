package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ComplianceOperatorControlRuleV2Edge --references=storage.ComplianceOperatorControlV2,storage.ComplianceOperatorRuleV2 --search-category=COMPLIANCE_CONTROL_RULE_EDGE
