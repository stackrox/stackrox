package internal

// Version variables with fallback defaults for ad-hoc builds (e.g. `go build`
// without the build infrastructure). When building via go-tool.sh, these are
// overridden by the generated zversion.go init() function.
var (
	// MainVersion is the Rox version.
	MainVersion string
	// CollectorVersion is the collector version to be used by default.
	CollectorVersion string
	// FactVersion is the fact version to be used by default.
	FactVersion string
	// ScannerVersion is the scanner version to be used with this Rox version.
	ScannerVersion string
	// GitShortSha is the short git commit SHA.
	GitShortSha string
)
