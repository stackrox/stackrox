package reportgenerator

import (
	"time"

	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/walker"
)

// ErrUserCancelled is the cause attached to a context when a user cancels a running report.
var ErrUserCancelled = errors.New("report cancelled by user")

// ReportRequest contains information needed to generate and notify a report
type ReportRequest struct {
	ReportSnapshot *storage.ReportSnapshot
	Collection     *storage.ResourceCollection
	DataStartTime  time.Time
}

type reportEmailBodyFormat struct {
	BrandedPrefix string
}

type reportEmailSubjectFormat struct {
	BrandedProductNameShort string
	ReportConfigName        string
	CollectionName          string
}

// ImageCVEQueryResponse contains the fields of report query response
type ImageCVEQueryResponse struct {
	Cluster           *string                        `db:"cluster"`
	Namespace         *string                        `db:"namespace"`
	Deployment        *string                        `db:"deployment"`
	Image             *string                        `db:"image"`
	Component         *string                        `db:"component"`
	ComponentVersion  *string                        `db:"component_version"`
	CVEID             *string                        `db:"cve_id"`
	CVE               *string                        `db:"cve"`
	Fixable           *bool                          `db:"fixable"`
	FixedByVersion    *string                        `db:"fixed_by"`
	Severity          *storage.VulnerabilitySeverity `db:"severity"`
	CVSS              *float64                       `db:"cvss"`
	NVDCVSS           *float64                       `db:"nvd_cvss"`
	EPSSProbability   *float64                       `db:"epss_probability"`
	CisaKev           *bool                          `db:"cisa_kev"`
	DiscoveredAtImage *time.Time                     `db:"first_image_occurrence_timestamp"`
	ImageCreatedAt    *time.Time                     `db:"image_created_time"`
	AdvisoryName      *string                        `db:"advisory_name"`
	AdvisoryLink      *string                        `db:"advisory_link"`
	Origin            *storage.VulnOrigin            `db:"cve_origin"`

	Link string
}

func (res *ImageCVEQueryResponse) GetCluster() string {
	if res.Cluster == nil {
		return ""
	}
	return *res.Cluster
}

func (res *ImageCVEQueryResponse) GetNamespace() string {
	if res.Namespace == nil {
		return ""
	}
	return *res.Namespace
}

func (res *ImageCVEQueryResponse) GetDeployment() string {
	if res.Deployment == nil {
		return ""
	}
	return *res.Deployment
}

func (res *ImageCVEQueryResponse) GetImage() string {
	if res.Image == nil {
		return ""
	}
	return *res.Image
}

func (res *ImageCVEQueryResponse) GetComponent() string {
	if res.Component == nil {
		return ""
	}
	return *res.Component
}

func (res *ImageCVEQueryResponse) GetComponentVersion() string {
	if res.ComponentVersion == nil {
		return ""
	}
	return *res.ComponentVersion
}

func (res *ImageCVEQueryResponse) GetCVEID() string {
	if res.CVEID == nil {
		return ""
	}
	return *res.CVEID
}

func (res *ImageCVEQueryResponse) GetCVE() string {
	if res.CVE == nil {
		return ""
	}
	return *res.CVE
}

func (res *ImageCVEQueryResponse) GetFixable() bool {
	if res.Fixable == nil {
		return false
	}
	return *res.Fixable
}

func (res *ImageCVEQueryResponse) GetFixedByVersion() string {
	if res.FixedByVersion == nil {
		return ""
	}
	return *res.FixedByVersion
}

func (res *ImageCVEQueryResponse) GetSeverity() storage.VulnerabilitySeverity {
	if res.Severity == nil {
		return storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY
	}
	return *res.Severity
}

func (res *ImageCVEQueryResponse) GetCVSS() float64 {
	if res.CVSS == nil {
		return 0.0
	}
	return *res.CVSS
}

func (res *ImageCVEQueryResponse) GetOrigin() storage.VulnOrigin {
	if res.Origin == nil {
		return storage.VulnOrigin_VULN_ORIGIN_OTHER
	}
	return *res.Origin
}

func (res *ImageCVEQueryResponse) GetNVDCVSS() float64 {
	if res.NVDCVSS == nil {
		return 0.0
	}
	return *res.NVDCVSS
}

func (res *ImageCVEQueryResponse) GetEPSSProbability() *float64 {
	return res.EPSSProbability
}

func (res *ImageCVEQueryResponse) GetCisaKev() *bool {
	return res.CisaKev
}

func (res *ImageCVEQueryResponse) GetAdvisoryName() string {
	if res.AdvisoryName == nil {
		return "Not Available"
	}
	return *res.AdvisoryName
}

func (res *ImageCVEQueryResponse) GetAdvisoryLink() string {
	if res.AdvisoryLink == nil {
		return ""
	}
	return *res.AdvisoryLink
}

func (res *ImageCVEQueryResponse) GetDiscoveredAtImage() string {
	if res.DiscoveredAtImage == nil {
		return "Not Available"
	}
	return res.DiscoveredAtImage.Format("January 02, 2006")
}

func (res *ImageCVEQueryResponse) GetImageCreatedAt() string {
	if res.ImageCreatedAt == nil {
		return "Not Available"
	}
	return res.ImageCreatedAt.Format("2006-01-02")
}

// ReportQueryParts contains the parts used to build the report query
type ReportQueryParts struct {
	Schema     *walker.Schema
	Selects    []*v1.QuerySelect
	Pagination *v1.QueryPagination
}

// ReportData contains the cve rows to be included in the report along with counts of deployed and watched image CVEs
type ReportData struct {
	CVEResponses            []*ImageCVEQueryResponse
	NumDeployedImageResults int
	NumWatchedImageResults  int
}
