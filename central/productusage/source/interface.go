package source

// SecuredUnitsSource interface provides methods to get usage metrics from a source.
type SecuredUnitsSource interface {
	GetNodeCount() int64
	GetCpuCapacity() int64
}
