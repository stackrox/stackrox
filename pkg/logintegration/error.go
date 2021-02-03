package logintegration

const (
	// ErrFeatureNotEnabled is an error message conveying kubernetes audit log collection feature is not enabled.
	ErrFeatureNotEnabled = "support for kubernetes audit event detection is not enabled"

	// ErrNotFound is an error message conveying requested log integration not found.
	ErrNotFound = "log integration configuration %s not found"
)
