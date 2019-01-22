package framework

// Status indicates the status of a compliance check run.
type Status int32

const (
	// FailStatus indicates that a compliance check failed.
	FailStatus Status = iota
	// PassStatus indicates that a compliance check passed.
	PassStatus
	// SkipStatus indicates that a compliance check was skipped as it was not applicable.
	SkipStatus
)
