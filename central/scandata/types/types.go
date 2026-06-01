package types

import (
	"encoding/json"
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

// AdvisoryJSON represents the JSON structure stored in the advisories JSONB column.
type AdvisoryJSON struct {
	ID       string  `json:"id"`
	Severity string  `json:"severity"`
	CVSS     float32 `json:"cvss"`
	Source   string  `json:"source"`
}

// ParseAdvisories parses the advisories JSONB string into structured data.
// Returns nil if the JSON is empty or invalid.
func ParseAdvisories(jsonStr string) []AdvisoryJSON {
	if jsonStr == "" || jsonStr == "[]" {
		return nil
	}
	var advisories []AdvisoryJSON
	if err := json.Unmarshal([]byte(jsonStr), &advisories); err != nil {
		return nil
	}
	return advisories
}

// GetPrimaryAdvisoryID returns the first advisory ID from the JSONB, or empty string.
func GetPrimaryAdvisoryID(jsonStr string) string {
	advisories := ParseAdvisories(jsonStr)
	if len(advisories) == 0 {
		return ""
	}
	return advisories[0].ID
}

// GetPrimarySourceName returns the first source name from the JSONB, or empty string.
func GetPrimarySourceName(jsonStr string) string {
	advisories := ParseAdvisories(jsonStr)
	if len(advisories) == 0 {
		return ""
	}
	return advisories[0].Source
}

// GetAllAdvisoryIDs extracts all advisory IDs from the JSONB.
func GetAllAdvisoryIDs(jsonStr string) []string {
	advisories := ParseAdvisories(jsonStr)
	if len(advisories) == 0 {
		return nil
	}
	ids := make([]string, len(advisories))
	for i, adv := range advisories {
		ids[i] = adv.ID
	}
	return ids
}
