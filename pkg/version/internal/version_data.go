package internal

import (
	"embed"
)

var (
	// MainVersion is the Rox version. Set to base tag via ldflags (//XDef:).
	// For builds, GetMainVersion() overrides with the full version from MAIN_VERSION.
	MainVersion string //XDef:STABLE_MAIN_VERSION
	// CollectorVersion is the collector version to be used by default.
	CollectorVersion string //XDef:STABLE_COLLECTOR_VERSION
	// FactVersion is the fact version to be used by default.
	FactVersion string //XDef:STABLE_FACT_VERSION
	// ScannerVersion is the scanner version to be used with this Rox version.
	ScannerVersion string //XDef:STABLE_SCANNER_VERSION
	// GitShortSha is the (short) Git SHA that was built.
	GitShortSha string
)

// Optional untracked files written by go-tool.sh for builds.
// The *_VERSION glob matches them when present; when absent
// (tests, fresh clone, go vet), it matches EMPTY_VERSION.
//
//go:embed *_VERSION
var versionFS embed.FS

// GetMainVersion returns the full version string. For builds, this is the
// detailed version from MAIN_VERSION (e.g. 4.7.0-123-gabcdef1234).
// For tests or when the file is absent, returns the ldflags value.
func GetMainVersion() string {
	if data, err := versionFS.ReadFile("MAIN_VERSION"); err == nil {
		return string(data)
	}
	return MainVersion
}

// GetGitShortSha returns the git short SHA from the embedded file,
// or the value set by test code if the file is absent.
func GetGitShortSha() string {
	if data, err := versionFS.ReadFile("GIT_SHORT_SHA_VERSION"); err == nil {
		return string(data)
	}
	return GitShortSha
}
