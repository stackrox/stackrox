package imagecve

import (
	"time"
)

type imageCVECore struct {
	CVE                         string    `db:"cve"`
	ImagesWithCriticalSeverity  int       `db:"images_with_critical_severity"`
	ImagesWithImportantSeverity int       `db:"images_with_important_severity"`
	ImagesWithModerateSeverity  int       `db:"images_with_moderate_severity"`
	ImagesWithLowSeverity       int       `db:"images_with_low_severity"`
	TopCVSS                     float32   `db:"cvss_max"`
	AffectedImages              int       `db:"image_sha_count"`
	FirstDiscoveredInSystem     time.Time `db:"cve_created_time_min"`
}

func (c *imageCVECore) GetCVE() string {
	return c.CVE
}

func (c *imageCVECore) GetImagesBySeverity() *ResourceCountByCVESeverity {
	return &ResourceCountByCVESeverity{
		CriticalSeverityCount:  c.ImagesWithCriticalSeverity,
		ImportantSeverityCount: c.ImagesWithImportantSeverity,
		ModerateSeverityCount:  c.ImagesWithModerateSeverity,
		LowSeverityCount:       c.ImagesWithLowSeverity,
	}
}

func (c *imageCVECore) GetTopCVSS() float32 {
	return c.TopCVSS
}

func (c *imageCVECore) GetAffectedImages() int {
	return c.AffectedImages
}

func (c *imageCVECore) GetFirstDiscoveredInSystem() time.Time {
	return c.FirstDiscoveredInSystem
}

type imageCVECoreCount struct {
	CVECount int `db:"cve_count"`
}
