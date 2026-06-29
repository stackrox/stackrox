package env

import "time"

var (
	// ComplianceV1MaxConcurrency limits how many V1 compliance operator profile/rule
	// updates can run concurrently. Each update acquires a DB connection and may Walk
	// the profiles table, so unbounded concurrency during sensor reconnects can exhaust
	// the connection pool.
	ComplianceV1MaxConcurrency = RegisterIntegerSetting("ROX_COMPLIANCE_V1_MAX_CONCURRENCY", 5).WithMinimum(1).WithMaximum(30)

	// ComplianceV1SemaphoreWaitTime is the maximum time a pipeline worker will wait
	// to acquire a semaphore slot before dropping the operation.
	ComplianceV1SemaphoreWaitTime = registerDurationSetting("ROX_COMPLIANCE_V1_SEMAPHORE_WAIT_TIME", 2*time.Minute, WithDurationZeroAllowed())
)
