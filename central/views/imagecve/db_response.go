package imagecve

import (
	"time"

	"github.com/stackrox/rox/central/views/common"
)

type imageCVECore struct {
	CVE                                string    `db:"cve"`
	CVEIDs                             []string  `db:"cve_id"`
	ImagesWithCriticalSeverity         int       `db:"critical_severity_count"`
	FixableImagesWithCriticalSeverity  int       `db:"fixable_critical_severity_count"`
	ImagesWithImportantSeverity        int       `db:"important_severity_count"`
	FixableImagesWithImportantSeverity int       `db:"fixable_important_severity_count"`
	ImagesWithModerateSeverity         int       `db:"moderate_severity_count"`
	FixableImagesWithModerateSeverity  int       `db:"fixable_moderate_severity_count"`
	ImagesWithLowSeverity              int       `db:"low_severity_count"`
	FixableImagesWithLowSeverity       int       `db:"fixable_low_severity_count"`
	TopCVSS                            float32   `db:"cvss_max"`
	AffectedImages                     int       `db:"image_sha_count"`
	FirstDiscoveredInSystem            time.Time `db:"cve_created_time_min"`
}

func (c *imageCVECore) GetCVE() string {
	return c.CVE
}

func (c *imageCVECore) GetCVEIDs() []string {
	return c.CVEIDs
}

func (c *imageCVECore) GetImagesBySeverity() common.ResourceCountByCVESeverity {
	return &resourceCountByImageCVESeverity{
		CriticalSeverityCount:         c.ImagesWithCriticalSeverity,
		FixableCriticalSeverityCount:  c.FixableImagesWithCriticalSeverity,
		ImportantSeverityCount:        c.ImagesWithImportantSeverity,
		FixableImportantSeverityCount: c.FixableImagesWithImportantSeverity,
		ModerateSeverityCount:         c.ImagesWithModerateSeverity,
		FixableModerateSeverityCount:  c.FixableImagesWithModerateSeverity,
		LowSeverityCount:              c.ImagesWithLowSeverity,
		FixableLowSeverityCount:       c.FixableImagesWithLowSeverity,
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

type resourceCountByFixability struct {
	total   int
	fixable int
}

func (r *resourceCountByFixability) GetTotal() int {
	return r.total
}

func (r *resourceCountByFixability) GetFixable() int {
	return r.fixable
}

type resourceCountByImageCVESeverity struct {
	CriticalSeverityCount         int `db:"critical_severity_count"`
	FixableCriticalSeverityCount  int `db:"fixable_critical_severity_count"`
	ImportantSeverityCount        int `db:"important_severity_count"`
	FixableImportantSeverityCount int `db:"fixable_important_severity_count"`
	ModerateSeverityCount         int `db:"moderate_severity_count"`
	FixableModerateSeverityCount  int `db:"fixable_moderate_severity_count"`
	LowSeverityCount              int `db:"low_severity_count"`
	FixableLowSeverityCount       int `db:"fixable_low_severity_count"`
}

func (r *resourceCountByImageCVESeverity) GetCriticalSeverityCount() common.ResourceCountByFixability {
	return &resourceCountByFixability{
		total:   r.CriticalSeverityCount,
		fixable: r.FixableCriticalSeverityCount,
	}
}

func (r *resourceCountByImageCVESeverity) GetImportantSeverityCount() common.ResourceCountByFixability {
	return &resourceCountByFixability{
		total:   r.ImportantSeverityCount,
		fixable: r.FixableImportantSeverityCount,
	}
}

func (r *resourceCountByImageCVESeverity) GetModerateSeverityCount() common.ResourceCountByFixability {
	return &resourceCountByFixability{
		total:   r.ModerateSeverityCount,
		fixable: r.FixableModerateSeverityCount,
	}
}

func (r *resourceCountByImageCVESeverity) GetLowSeverityCount() common.ResourceCountByFixability {
	return &resourceCountByFixability{
		total:   r.LowSeverityCount,
		fixable: r.FixableLowSeverityCount,
	}
}
