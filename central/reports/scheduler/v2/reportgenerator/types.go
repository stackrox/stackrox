package reportgenerator

import (
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
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

type DeployedImageResponse struct {
	Cluster        string                        `db:"cluster"`
	Namespace      string                        `db:"namespace"`
	Deployment     string                        `db:"deployment"`
	Image          string                        `db:"image"`
	Component      string                        `db:"component"`
	CVE            string                        `db:"cve"`
	Fixable        bool                          `db:"fixable"`
	FixedByVersion string                        `db:"fixed_by"`
	Severity       storage.VulnerabilitySeverity `db:"severity"`
	CVSS           float64                       `db:"cvss"`
	DiscoveredAt   *time.Time                    `db:"first_image_occurrence_timestamp"`
}
