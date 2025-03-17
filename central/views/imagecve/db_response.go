package imagecve

import (
	"time"

	"github.com/stackrox/rox/central/views/common"
	"github.com/stackrox/rox/generated/storage"
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
	TopCVSS                            *float32   `db:"cvss_max"`
	AffectedImageCount                 int        `db:"image_sha_count"`
	FirstDiscoveredInSystem            *time.Time `db:"cve_created_time_min"`
	Published                          *time.Time `db:"cve_published_on_min"`
	TopNVDCVSS                         *float32   `db:"nvd_cvss_max"`
}

func (c *imageCVECoreResponse) GetCVE() string {
	return c.CVE
}

func (c *imageCVECoreResponse) GetCVEIDs() []string {
	return c.CVEIDs
}

func (c *imageCVECoreResponse) GetImagesBySeverity() common.ResourceCountByCVESeverity {
	return &common.ResourceCountByImageCVESeverity{
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
	if c.TopCVSS == nil {
		return 0.0
	}
	return *c.TopCVSS
}

func (c *imageCVECoreResponse) GetTopNVDCVSS() float32 {
	if c.TopNVDCVSS == nil {
		return 0.0
	}
	return *c.TopNVDCVSS
}

func (c *imageCVECoreResponse) GetAffectedImageCount() int {
	return c.AffectedImageCount
}

func (c *imageCVECoreResponse) GetFirstDiscoveredInSystem() *time.Time {
	return c.FirstDiscoveredInSystem
}

func (c *imageCVECoreResponse) GetPublishDate() *time.Time {
	return c.Published
}

type imageCVECoreCount struct {
	CVECount int `db:"cve_count"`
}

type imageResponse struct {
	ImageID string `db:"image_sha"`
}

type deploymentResponse struct {
	DeploymentID string `db:"deployment_id"`
}

type imageCVEResponse struct {
	CVE                     string                         `db:"cve"`
	CVEIDs                  []string                       `db:"cve_id"`
	ComponentIDs            []string                       `db:"component_id"`
	Severity                *storage.VulnerabilitySeverity `db:"severity"`
	TopCVSS                 *float32                       `db:"cvss_max"`
	CVEState                *storage.VulnerabilityState    `db:"state"`
	AffectedComponentCount  int                            `db:"componentid_count"`
	FirstDiscoveredInSystem *time.Time                     `db:"cve_created_time_min"`
	Published               *time.Time                     `db:"cve_published_on_min"`
	TopNVDCVSS              *float32                       `db:"nvd_cvss_max"`
}

func (c *imageCVEResponse) GetCVE() string {
	return c.CVE
}

func (c *imageCVEResponse) GetCVEIDs() []string {
	return c.CVEIDs
}

func (c *imageCVEResponse) GetComponentIDs() []string {
	return c.ComponentIDs
}

func (c *imageCVEResponse) GetSeverity() *storage.VulnerabilitySeverity {
	return c.Severity
}

func (c *imageCVEResponse) GetTopCVSS() float32 {
	if c.TopCVSS == nil {
		return 0.0
	}
	return *c.TopCVSS
}

func (c *imageCVEResponse) GetTopNVDCVSS() float32 {
	if c.TopNVDCVSS == nil {
		return 0.0
	}
	return *c.TopNVDCVSS
}

func (c *imageCVEResponse) GetAffectedComponentCount() int {
	return len(c.ComponentIDs)
}

func (c *imageCVEResponse) GetFirstDiscoveredInSystem() *time.Time {
	return c.FirstDiscoveredInSystem
}

func (c *imageCVEResponse) GetPublishDate() *time.Time {
	return c.Published
}
