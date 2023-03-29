package imagecve

import (
	"time"

	"github.com/stackrox/rox/central/views/common"
)

type imageCVECore struct {
	CVE                         string    `db:"cve"`
	ImagesWithCriticalSeverity  int       `db:"critical_severity_count"`
	ImagesWithImportantSeverity int       `db:"important_severity_count"`
	ImagesWithModerateSeverity  int       `db:"moderate_severity_count"`
	ImagesWithLowSeverity       int       `db:"low_severity_count"`
	TopCVSS                     float32   `db:"cvss_max"`
	AffectedImages              int       `db:"image_sha_count"`
	FirstDiscoveredInSystem     time.Time `db:"cve_created_time_min"`
}

func (c *imageCVECore) GetCVE() string {
	return c.CVE
}

func (c *imageCVECore) GetImagesBySeverity() common.ResourceCountByCVESeverity {
	return &resourceCountByImageCVESeverity{
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

type resourceCountByImageCVESeverity struct {
	CriticalSeverityCount  int `db:"critical_severity_count"`
	ImportantSeverityCount int `db:"important_severity_count"`
	ModerateSeverityCount  int `db:"moderate_severity_count"`
	LowSeverityCount       int `db:"low_severity_count"`
}

func (r *resourceCountByImageCVESeverity) GetCriticalSeverityCount() int {
	return r.CriticalSeverityCount
}

func (r *resourceCountByImageCVESeverity) GetImportantSeverityCount() int {
	return r.ImportantSeverityCount
}

func (r *resourceCountByImageCVESeverity) GetModerateSeverityCount() int {
	return r.ModerateSeverityCount
}

func (r *resourceCountByImageCVESeverity) GetLowSeverityCount() int {
	return r.LowSeverityCount
}
