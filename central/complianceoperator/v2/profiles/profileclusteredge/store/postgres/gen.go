package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ComplianceOperatorProfileClusterEdge --references=storage.ComplianceOperatorProfileV2,storage.Cluster --search-category COMPLIANCE_PROFILE_CLUSTER_EDGE --feature-flag ComplianceEnhancements
