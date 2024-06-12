package postgres

//go:generate pg-table-bindings-wrapper --type=storage.VulnerabilityRequest --search-category VULN_REQUEST --permission-checker sac.NewAnyGlobalResourceAllowedPermissionChecker(resources.VulnerabilityManagementRequests,resources.VulnerabilityManagementApprovals)
