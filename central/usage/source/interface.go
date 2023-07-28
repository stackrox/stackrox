package source

// UsageSource interface provides methods to get usage metrics from a source.
//
//go:generate mockgen-wrapper
type UsageSource interface {
	GetNodeCount() int64
	GetCpuCapacity() int64
}
