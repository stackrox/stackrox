package internal

var (
	// BaseVersion is the base product version (e.g. "4.11.x"), read from the VERSION file.
	// Used by vcs.go to construct MainVersion at runtime when not set explicitly.
	BaseVersion string
	// MainVersion is the Rox version. For release builds, set directly by generate-version.sh
	// via BUILD_TAG. For dev builds, derived at runtime in vcs.go from BaseVersion + buildvcs.
	MainVersion string
	// CollectorVersion is the collector version to be used by default.
	CollectorVersion string
	// FactVersion is the fact version to be used by default.
	FactVersion string
	// ScannerVersion is the scanner version to be used with this Rox version.
	ScannerVersion string
	// GitShortSha is the (short) Git SHA that was built. Derived from buildvcs at runtime.
	GitShortSha string
)
