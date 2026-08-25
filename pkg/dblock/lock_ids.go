package dblock

// Advisory lock IDs are random int64 values generated via crypto/rand.
// New IDs must not collide with existing ones.
const (
	PruningGCLockID       int64 = 7_293_581_046_138_294_017
	ReportSchedulerLockID int64 = 4_618_739_250_183_647_293
)
