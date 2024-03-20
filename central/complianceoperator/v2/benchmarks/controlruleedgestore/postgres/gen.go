package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ComplianceOperatorControlRuleV2Edge --references=storage.ComplianceOperatorControl,references.ComplianceOperatorRuleV2
