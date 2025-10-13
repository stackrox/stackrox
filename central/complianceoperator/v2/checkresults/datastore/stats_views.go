package datastore

import (
	"time"
)

// ResourceResultCountByClusterScan represents shape of the stats query for compliance operator results
type ResourceResultCountByClusterScan struct {
	PassCount          int    `db:"compliance_pass_count"`
	FailCount          int    `db:"compliance_fail_count"`
	ErrorCount         int    `db:"compliance_error_count"`
	InfoCount          int    `db:"compliance_info_count"`
	ManualCount        int    `db:"compliance_manual_count"`
	NotApplicableCount int    `db:"compliance_not_applicable_count"`
	InconsistentCount  int    `db:"compliance_inconsistent_count"`
	ClusterID          string `db:"cluster_id"`
	ClusterName        string `db:"cluster"`
	ScanConfigName     string `db:"compliance_scan_config_name"`
}

// ResultStatusCountByCluster represents shape of the stats query for compliance operator results
// grouped by cluster
type ResultStatusCountByCluster struct {
	PassCount          int        `db:"compliance_pass_count"`
	FailCount          int        `db:"compliance_fail_count"`
	ErrorCount         int        `db:"compliance_error_count"`
	InfoCount          int        `db:"compliance_info_count"`
	ManualCount        int        `db:"compliance_manual_count"`
	NotApplicableCount int        `db:"compliance_not_applicable_count"`
	InconsistentCount  int        `db:"compliance_inconsistent_count"`
	ClusterID          string     `db:"cluster_id"`
	ClusterName        string     `db:"cluster"`
	LastScanTime       *time.Time `db:"compliance_scan_last_executed_time_max"`
}

type clusterCount struct {
	TotalCount int `db:"cluster_id_count"`
}

type profileCount struct {
	TotalCount int `db:"compliance_profile_name_count"`
}

type complianceCheckCount struct {
	TotalCount int `db:"compliance_check_name_count"`
}

type configurationCount struct {
	TotalCount int `db:"compliance_scan_config_name_count"`
}

// ResourceResultCountByProfile represents shape of the stats query for compliance operator results
type ResourceResultCountByProfile struct {
	PassCount          int    `db:"compliance_pass_count"`
	FailCount          int    `db:"compliance_fail_count"`
	ErrorCount         int    `db:"compliance_error_count"`
	InfoCount          int    `db:"compliance_info_count"`
	ManualCount        int    `db:"compliance_manual_count"`
	NotApplicableCount int    `db:"compliance_not_applicable_count"`
	InconsistentCount  int    `db:"compliance_inconsistent_count"`
	ProfileName        string `db:"compliance_profile_name"`
}

// ResourceResultsByProfile represents shape of the stats query for compliance operator results
type ResourceResultsByProfile struct {
	PassCount          int    `db:"compliance_pass_count"`
	FailCount          int    `db:"compliance_fail_count"`
	ErrorCount         int    `db:"compliance_error_count"`
	InfoCount          int    `db:"compliance_info_count"`
	ManualCount        int    `db:"compliance_manual_count"`
	NotApplicableCount int    `db:"compliance_not_applicable_count"`
	InconsistentCount  int    `db:"compliance_inconsistent_count"`
	ProfileName        string `db:"compliance_profile_name"`
	CheckName          string `db:"compliance_check_name"`
	RuleName           string `db:"compliance_rule_name"`
	CheckRationale     string `db:"compliance_check_rationale"`
}
