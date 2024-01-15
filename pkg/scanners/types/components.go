package types

import (
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	v1 "github.com/stackrox/scanner/generated/scanner/api/v1"
)

// ScanComponents holds the different types of components (a.k.a index reports)
// used by scanners when performing vulnerability matching.
type ScanComponents struct {
	v1comps        *v1.Components
	v4comps        *v4.Contents
	indexerVersion string
}

// NewScanComponents creates a new ScanComponents.
func NewScanComponents(indexerVersion string, v1comps *v1.Components, v4comps *v4.Contents) *ScanComponents {
	return &ScanComponents{
		v1comps:        v1comps,
		v4comps:        v4comps,
		indexerVersion: indexerVersion,
	}
}

// Clairify returns components used by the Clairify scanner.
func (s *ScanComponents) Clairify() *v1.Components {
	return s.v1comps
}

// ScannerV4 returns components used by the Scanner V4 scanner.
func (s *ScanComponents) ScannerV4() *v4.Contents {
	return s.v4comps
}

// ScannerType returns the scanner type for which components
// have been populated for.
func (s *ScanComponents) ScannerType() string {
	if ScannerV4IndexerVersion(s.indexerVersion) {
		return ScannerV4
	}

	return Clairify
}

// ScannerV4IndexerVersion returns true if version represents a ScannerV4 indexer.
func ScannerV4IndexerVersion(version string) bool {
	// If indexer version is NOT empty, then assume the version represents a
	// valid Scanner V4 indexer.
	return version != ""
}
