package types

import (
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	v1 "github.com/stackrox/scanner/generated/scanner/api/v1"
)

type ScanComponents struct {
	v1comps        *v1.Components
	v4comps        *v4.Contents
	indexerVersion string
}

func NewScanComponents(indexerVersion string, v1comps *v1.Components, v4comps *v4.Contents) *ScanComponents {
	return &ScanComponents{
		v1comps:        v1comps,
		v4comps:        v4comps,
		indexerVersion: indexerVersion,
	}
}

func (s *ScanComponents) Clairify() *v1.Components {
	return s.v1comps
}

func (s *ScanComponents) ScannerV4() *v4.Contents {
	return s.v4comps
}

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
