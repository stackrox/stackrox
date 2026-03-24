package internal

// Version variables are populated at init time by the generated zversion.go
// file (created by go-tool.sh). Without go-tool.sh, all values remain empty.
var (
	// MainVersion is the Rox version.
	MainVersion string
	// CollectorVersion is the collector version to be used by default.
	CollectorVersion string
	// FactVersion is the fact version to be used by default.
	FactVersion string
	// ScannerVersion is the scanner version to be used with this Rox version.
	ScannerVersion string
	// GitShortSha is the (short) Git SHA that was built.
	GitShortSha string
)
