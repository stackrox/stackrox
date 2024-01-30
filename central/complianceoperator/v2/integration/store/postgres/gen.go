package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ComplianceIntegration --search-category COMPLIANCE_INTEGRATIONS --references=storage.Cluster --search-scope CLUSTERS
