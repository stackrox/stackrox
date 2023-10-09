package reportgenerator

import (
	"time"

	"github.com/gogo/protobuf/types"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/walker"
)

// ReportRequest contains information needed to generate and notify a report
type ReportRequest struct {
	ReportSnapshot *storage.ReportSnapshot
	Collection     *storage.ResourceCollection
	DataStartTime  *types.Timestamp
}

type reportEmailBodyFormat struct {
	BrandedProductName      string
	BrandedProductNameShort string
}

type reportEmailSubjectFormat struct {
	BrandedProductNameShort string
	ReportConfigName        string
	CollectionName          string
}

// ImageCVEQueryResponse contains the fields of report query response
type ImageCVEQueryResponse struct {
	Cluster           string                        `db:"cluster"`
	Namespace         string                        `db:"namespace"`
	Deployment        string                        `db:"deployment"`
	Image             string                        `db:"image"`
	Component         string                        `db:"component"`
	ComponentVersion  string                        `db:"component_version"`
	CVEID             string                        `db:"cve_id"`
	CVE               string                        `db:"cve"`
	Fixable           bool                          `db:"fixable"`
	FixedByVersion    string                        `db:"fixed_by"`
	Severity          storage.VulnerabilitySeverity `db:"severity"`
	CVSS              float64                       `db:"cvss"`
	DiscoveredAtImage *time.Time                    `db:"first_image_occurrence_timestamp"`

	Link string
}

// ReportQueryParams contains the parts used to build the report query
type ReportQueryParams struct {
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
