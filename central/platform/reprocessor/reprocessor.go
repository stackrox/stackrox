package reprocessor

// PlatformReprocessor reprocesses alerts and deployments to mark those that are platform components
//
//go:generate mockgen-wrapper
type PlatformReprocessor interface {
	// Start PlatformReprocessor. Can only have one instance running at a time.
	Start()

	// Stop PlatformReprocessor
	Stop()
	RunReprocessor()
}
