package imagecveflat

import (
	"time"
)

type imageCVEFlatResponse struct {
	CVE                     string     `db:"cve"`
	CVEIDs                  []string   `db:"cve_id"`
	Severity                int        `db:"severity_max"`
	TopCVSS                 *float32   `db:"cvss_max"`
	AffectedImageCount      int        `db:"image_sha_count"`
	FirstDiscoveredInSystem *time.Time `db:"cve_created_time_min"`
	Published               *time.Time `db:"cve_published_on_min"`
	TopNVDCVSS              *float32   `db:"nvd_cvss_max"`
	EpssProbability         *float32   `db:"cvebaseinfo_epss_epssprobability_max"`
	ImpactScore             *float32   `db:"impactscore_max"`
	FirstImageOccurrence    *time.Time `db:"firstimageoccurrence_min"`
	State                   int        `db:"state_max"`
	Fixable                 bool       `db:"isfixable_max"`
}

func (c *imageCVEFlatResponse) GetCVE() string {
	return c.CVE
}

func (c *imageCVEFlatResponse) GetCVEIDs() []string {
	return c.CVEIDs
}

func (c *imageCVEFlatResponse) GetSeverity() int {
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

func (c *imageCVEFlatResponse) GetState() int { return c.State }

func (c *imageCVEFlatResponse) IsFixable() bool { return c.Fixable }

type imageCVEFlatCount struct {
	CVECount int `db:"cve_count"`
}
