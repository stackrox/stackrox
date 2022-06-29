package rocksdb

//go:generate rocksdb-bindings-wrapper --type=ComplianceOperatorRule --bucket=complianceoperatorrules --cache --migrate-seq 10 --migrate-to compliance_operator_rules
