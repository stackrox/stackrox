package imagecveflat

import (
	"time"

	"github.com/stackrox/rox/generated/storage"
)

type imageCVEFlatResponse struct {
	CVE                     string                         `db:"cve"`
	CVEIDs                  []string                       `db:"cve_id"`
	Severity                *storage.VulnerabilitySeverity `db:"severity_max"`
	TopCVSS                 *float32                       `db:"cvss_max"`
	AffectedImageCount      int                            `db:"image_sha_count"`
	FirstDiscoveredInSystem *time.Time                     `db:"cve_created_time_min"`
	Published               *time.Time                     `db:"cve_published_on_min"`
	TopNVDCVSS              *float32                       `db:"nvd_cvss_max"`
	EpssProbability         *float32                       `db:"cvebaseinfo_epss_epssprobability_max"`
	ImpactScore             *float32                       `db:"impactscore_max"`
	FirstImageOccurrence    *time.Time                     `db:"firstimageoccurrence_min"`
	CreatedAt               *time.Time                     `db:"created_at_min"`
	State                   *storage.VulnerabilityState    `db:"state_max"`
}

func (c *imageCVEFlatResponse) GetCVE() string {
	return c.CVE
}

func (c *imageCVEFlatResponse) GetCVEIDs() []string {
	return c.CVEIDs
}

func (c *imageCVEFlatResponse) GetSeverity() *storage.VulnerabilitySeverity {
	return c.Severity
}

func (c *imageCVEFlatResponse) GetTopCVSS() float32 {
	if c.TopCVSS == nil {
		return 0.0
	}
	return *c.TopCVSS
}

func (c *imageCVEFlatResponse) GetTopNVDCVSS() float32 {
	if c.TopNVDCVSS == nil {
		return 0.0
	}
	return *c.TopNVDCVSS
}

func (c *imageCVEFlatResponse) GetAffectedImageCount() int {
	return c.AffectedImageCount
}

func (c *imageCVEFlatResponse) GetFirstDiscoveredInSystem() *time.Time {
	return c.FirstDiscoveredInSystem
}

func (c *imageCVEFlatResponse) GetPublishDate() *time.Time {
	return c.Published
}

func (c *imageCVEFlatResponse) GetFirstImageOccurrence() *time.Time {
	return c.FirstImageOccurrence
}

func (c *imageCVEFlatResponse) GetCreatedAt() *time.Time {
	return c.CreatedAt
}

func (c *imageCVEFlatResponse) GetState() *storage.VulnerabilityState { return c.State }

type imageCVEFlatCount struct {
	CVECount int `db:"cve_count"`
}
