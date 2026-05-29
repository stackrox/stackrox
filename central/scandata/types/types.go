package types

import (
	"time"

	"github.com/stackrox/rox/generated/storage"
)

// ScanData represents a complete scan result: scan metadata + components + findings
type ScanData struct {
	Scan       *storage.ImageScanV2
	Components []*storage.ScanComponent
	Findings   []*storage.ScanFinding
}

// CVEListRow represents one row in the CVE list page
type CVEListRow struct {
	CVEName    string
	Severity   int32
	CVSS       float32
	ImageCount int
	Fixable    bool
	FirstSeen  *time.Time
}

// FindingWithComponent is a finding joined with its parent component's metadata.
type FindingWithComponent struct {
	Finding          *storage.ScanFinding
	ComponentName    string
	ComponentVersion string
	ComponentSource  int32
}
