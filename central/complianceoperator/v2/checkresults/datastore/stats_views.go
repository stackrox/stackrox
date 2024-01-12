package datastore

// ResourceCountByResultByCluster represents shape of the stats query for compliance operator results grouped
// by cluster and scan configuration
type ResourceCountByResultByCluster struct {
	PassCount          int    `db:"pass_count"`
	FailCount          int    `db:"fail_count"`
	ErrorCount         int    `db:"error_count"`
	InfoCount          int    `db:"info_count"`
	ManualCount        int    `db:"manual_count"`
	NotApplicableCount int    `db:"not_applicable_count"`
	InconsistentCount  int    `db:"inconsistent_count"`
	ClusterID          string `db:"cluster_id"`
	ClusterName        string `db:"cluster"`
	ScanConfigName     string `db:"compliance_scan_config_name"`
}

// ResultStatusCountByCheckResult represents shape of the stats query for compliance operator results
// grouped by check result name
type ResultStatusCountByCheckResult struct {
	PassCount          int    `db:"pass_count"`
	FailCount          int    `db:"fail_count"`
	ErrorCount         int    `db:"error_count"`
	InfoCount          int    `db:"info_count"`
	ManualCount        int    `db:"manual_count"`
	NotApplicableCount int    `db:"not_applicable_count"`
	InconsistentCount  int    `db:"inconsistent_count"`
	CheckName          string `db:"check_name"`
	CheckDescription   string `db:"description"`
}
