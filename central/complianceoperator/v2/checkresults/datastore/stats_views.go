package datastore

// ResourceResultCountByClusterScan represents shape of the stats query for compliance operator results
type ResourceResultCountByClusterScan struct {
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

// ResultStatusCountByCluster represents shape of the stats query for compliance operator results
// grouped by cluster
type ResultStatusCountByCluster struct {
	PassCount          int    `db:"pass_count"`
	FailCount          int    `db:"fail_count"`
	ErrorCount         int    `db:"error_count"`
	InfoCount          int    `db:"info_count"`
	ManualCount        int    `db:"manual_count"`
	NotApplicableCount int    `db:"not_applicable_count"`
	InconsistentCount  int    `db:"inconsistent_count"`
	ClusterID          string `db:"cluster_id"`
	ClusterName        string `db:"cluster"`
}

type clusterStatsCount struct {
	ClusterCount int `db:"cluster_id_count"`
}

// ResourceResultCountByProfile represents shape of the stats query for compliance operator results
type ResourceResultCountByProfile struct {
	PassCount          int    `db:"pass_count"`
	FailCount          int    `db:"fail_count"`
	ErrorCount         int    `db:"error_count"`
	InfoCount          int    `db:"info_count"`
	ManualCount        int    `db:"manual_count"`
	NotApplicableCount int    `db:"not_applicable_count"`
	InconsistentCount  int    `db:"inconsistent_count"`
	ProfileName        string `db:"compliance_profile_name"`
}
