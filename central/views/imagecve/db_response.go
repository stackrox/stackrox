package imagecve

import (
	"time"

	"github.com/stackrox/rox/central/views/common"
)

type imageCVECoreResponse struct {
	CVE                                string     `db:"cve"`
	CVEIDs                             []string   `db:"cve_id"`
	ImagesWithCriticalSeverity         int        `db:"critical_severity_count"`
	FixableImagesWithCriticalSeverity  int        `db:"fixable_critical_severity_count"`
	ImagesWithImportantSeverity        int        `db:"important_severity_count"`
	FixableImagesWithImportantSeverity int        `db:"fixable_important_severity_count"`
	ImagesWithModerateSeverity         int        `db:"moderate_severity_count"`
	FixableImagesWithModerateSeverity  int        `db:"fixable_moderate_severity_count"`
	ImagesWithLowSeverity              int        `db:"low_severity_count"`
	FixableImagesWithLowSeverity       int        `db:"fixable_low_severity_count"`
	TopCVSS                            float32    `db:"cvss_max"`
	AffectedImageCount                 int        `db:"image_sha_count"`
	FirstDiscoveredInSystem            *time.Time `db:"cve_created_time_min"`
}

func (c *imageCVECoreResponse) GetCVE() string {
	return c.CVE
}

func (c *imageCVECoreResponse) GetCVEIDs() []string {
	return c.CVEIDs
}

func (c *imageCVECoreResponse) GetImagesBySeverity() common.ResourceCountByCVESeverity {
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

func (c *imageCVECoreResponse) GetTopCVSS() float32 {
	return c.TopCVSS
}

func (c *imageCVECoreResponse) GetAffectedImageCount() int {
	return c.AffectedImageCount
}

func (c *imageCVECoreResponse) GetFirstDiscoveredInSystem() *time.Time {
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

type imageResponse struct {
	ImageID string `db:"image_sha"`

	// Following are supported sort options.
	ImageFullName   string     `db:"image"`
	OperatingSystem string     `db:"image_os"`
	ScanTime        *time.Time `db:"image_scan_time"`
}

type deploymentResponse struct {
	DeploymentID string `db:"deployment_id"`

	// Following are supported sort options.
	DeploymentName string `db:"deployment"`
	Cluster        string `db:"cluster"`
	Namespace      string `db:"namespace"`
}
