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
	CVEName         string
	Severity        int32
	CVSS            float32
	ImageCount      int
	Fixable         bool
	FirstSeen       *time.Time
	PublishedDate   *time.Time
	EPSSProbability float32
}

// ImageBasicInfo holds the UUID and display name for an image looked up by digest.
type ImageBasicInfo struct {
	UUID     string
	FullName string
}

// FindingWithComponent is a finding joined with its parent component's metadata.
type FindingWithComponent struct {
	Finding           *storage.ScanFinding
	ComponentName     string
	ComponentVersion  string
	ComponentSource   int32
	ComponentLocation string
	ComponentArch     string
}

// DeploymentListRow represents one deployment in the list page.
type DeploymentListRow struct {
	ID          string
	Name        string
	ClusterID   string
	ClusterName string
	Namespace   string
	ImageCount  int
	CVECount    int
	TopSeverity int32
	Fixable     bool
}

// DeploymentImageRow represents one image in a deployment's detail view.
type DeploymentImageRow struct {
	ImageID     string
	ImageUUID   string
	ImageName   string
	CVECount    int
	TopSeverity int32
	Fixable     bool
}

// AdvisoryListRow represents one advisory in the advisory list page.
type AdvisoryListRow struct {
	AdvisoryID  string
	CVEName     string
	Severity    int32
	CVSS        float32
	SourceName  string
	Description string
	FixedBy     string
	ImageCount  int
}

// ComponentListRow represents one row in the component list page.
type ComponentListRow struct {
	Name           string
	VersionCount   int
	CVECount       int
	ImageCount     int
	TopSeverity    int32
	TopCVSS        float32
	CriticalCount  int
	ImportantCount int
	ModerateCount  int
	LowCount       int
}

// ComponentImageRow represents one image containing a given component.
type ComponentImageRow struct {
	ImageID     string
	ImageUUID   string
	ImageName   string
	Version     string
	Arch        string
	CVECount    int
	TopSeverity int32
	Fixable     bool
}

// ImageListRow represents one row in the image list page.
type ImageListRow struct {
	ImageID        string
	CVECount       int
	ComponentCount int
	TopSeverity    int32
	TopCVSS        float32
	Fixable        bool
	ScanTime       *time.Time
	CriticalCount  int
	ImportantCount int
	ModerateCount  int
	LowCount       int
}

// ComponentVersionInfo represents one version of a component with CVE data.
type ComponentVersionInfo struct {
	Version     string
	Source      string
	Arch        string
	Module      string
	CVECount    int
	ImageCount  int
	TopSeverity int32
	TopCVSS     float32
	Fixable     bool
	FixedBy     string
}
