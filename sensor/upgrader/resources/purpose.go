package resources

// Purpose declares the purpose of a resource (bundle or state).
//
//go:generate stringer -type=Purpose
type Purpose int32

// The following block enumerates all values for resource purposes.
const (
	StateResource Purpose = 1 << iota
	BundleResource
)
