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
	// StaleFreshnessCount and CurrentFreshnessCount are counts, over the cluster's
	// results, of rows with STALE and CURRENT stored data_freshness respectively
	// (UNKNOWN rows are counted by neither). Together they derive the v2 aggregate
	// freshness: INCONSISTENT when both counts are positive (a genuine mix of
	// stale and current data), STALE when only StaleFreshnessCount is positive,
	// CURRENT when only CurrentFreshnessCount is positive, else UNKNOWN. See
	// storagetov2.DeriveAggregateDataFreshness.
	StaleFreshnessCount   int `db:"compliance_data_freshness_stale_count"`
	CurrentFreshnessCount int `db:"compliance_data_freshness_current_count"`
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

// ProfileStaleStats holds the profile-scoped stale-data banner counts: how many
// clusters have any stale result and how many stale check results exist, across
// all clusters (not limited to a page).
type ProfileStaleStats struct {
	StaleClusterCount     int
	StaleCheckResultCount int
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
	// StaleFreshnessCount and CurrentFreshnessCount are counts, across all
	// clusters reporting this check, of rows with STALE and CURRENT stored
	// data_freshness respectively (UNKNOWN rows are counted by neither).
	// Together they derive the v2 aggregate freshness: INCONSISTENT when both
	// counts are positive (a genuine mix of stale and current data), STALE when
	// only StaleFreshnessCount is positive, CURRENT when only
	// CurrentFreshnessCount is positive, else UNKNOWN. See
	// storagetov2.DeriveAggregateDataFreshness.
	StaleFreshnessCount   int `db:"compliance_data_freshness_stale_count"`
	CurrentFreshnessCount int `db:"compliance_data_freshness_current_count"`
}
