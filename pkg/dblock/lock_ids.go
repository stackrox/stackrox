package dblock

// Advisory lock IDs are random int64 values generated via crypto/rand.
// New IDs must not collide with existing ones.
const (
	MigrationLockID           int64 = 7_517_845_236_103_920_641
	BackgroundMigrationLockID int64 = 2_846_193_750_482_637_519
	PruningGCLockID           int64 = 7_293_581_046_138_294_017
	ReportSchedulerLockID     int64 = 4_618_739_250_183_647_293
	// ComplianceReportSchedulerLockID guards the compliance report scheduler
	// (settle-audit-snapshot reporting, ROX_COMPLIANCE_REPORTS_OWN_SCHEDULER)
	// so that only one Central instance fires scheduled reports.
	ComplianceReportSchedulerLockID int64 = 5_812_647_930_451_762_084
)
