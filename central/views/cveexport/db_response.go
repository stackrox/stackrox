package cveexport

import (
	"time"

	"github.com/stackrox/rox/generated/storage"
)

type cveExportResponse struct {
	CVE             string                         `db:"cve"`
	CVEIDs          []string                       `db:"cve_id"`
	Severity        *storage.VulnerabilitySeverity `db:"severity_max"`
	CVSS            *float32                       `db:"cvss_max"`
	NVDCVSS         *float32                       `db:"nvd_cvss_max"`
	Summary         *string                        `db:"cve_summary_min"`
	Link            *string                        `db:"cve_link_min"`
	PublishedOn     *time.Time                     `db:"cve_published_on_min"`
	EPSSProbability *float32                       `db:"epss_probability_max"`
	EPSSPercentile  *float32                       `db:"epss_percentile_max"`
	AdvisoryName    *string                        `db:"advisory_name_min"`
	AdvisoryLink    *string                        `db:"advisory_link_min"`
}

func (c *cveExportResponse) GetCVE() string {
	return c.CVE
}

func (c *cveExportResponse) GetCVEIDs() []string {
	return c.CVEIDs
}

func (c *cveExportResponse) GetSeverity() storage.VulnerabilitySeverity {
	if c.Severity == nil {
		return storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY
	}
	return *c.Severity
}

func (c *cveExportResponse) GetCVSS() float32 {
	if c.CVSS == nil {
		return 0
	}
	return *c.CVSS
}

func (c *cveExportResponse) GetNVDCVSS() float32 {
	if c.NVDCVSS == nil {
		return 0
	}
	return *c.NVDCVSS
}

func (c *cveExportResponse) GetSummary() string {
	if c.Summary == nil {
		return ""
	}
	return *c.Summary
}

func (c *cveExportResponse) GetLink() string {
	if c.Link == nil {
		return ""
	}
	return *c.Link
}

func (c *cveExportResponse) GetPublishedOn() *time.Time {
	return c.PublishedOn
}

func (c *cveExportResponse) GetEPSSProbability() float32 {
	if c.EPSSProbability == nil {
		return 0
	}
	return *c.EPSSProbability
}

func (c *cveExportResponse) GetEPSSPercentile() float32 {
	if c.EPSSPercentile == nil {
		return 0
	}
	return *c.EPSSPercentile
}

func (c *cveExportResponse) GetAdvisoryName() string {
	if c.AdvisoryName == nil {
		return ""
	}
	return *c.AdvisoryName
}

func (c *cveExportResponse) GetAdvisoryLink() string {
	if c.AdvisoryLink == nil {
		return ""
	}
	return *c.AdvisoryLink
}

type cveCountResponse struct {
	CVECount int `db:"cve_count"`
}
