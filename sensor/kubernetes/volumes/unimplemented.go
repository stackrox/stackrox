package volumes

// Unimplemented is the default type for a volume that is unknown
type Unimplemented struct{}

// Source returns the source of the specific implementation
func (*Unimplemented) Source() string { return "" }

// Type returns the type of the volume which matches up with the struct name inside Kubernetes
func (*Unimplemented) Type() string { return "unknown" }
