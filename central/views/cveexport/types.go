package cveexport

import (
	"context"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// CveExport represents a single CVE aggregated across all images and components.
type CveExport interface {
	GetCVE() string
	GetSeverity() storage.VulnerabilitySeverity
	GetCVSS() float32
	GetNVDCVSS() float32
	GetSummary() string
	GetLink() string
	GetPublishedOn() *time.Time
	GetEPSSProbability() float32
	GetEPSSPercentile() float32
	GetAdvisoryName() string
	GetAdvisoryLink() string
	GetCVEIDs() []string
}

// CveExportView provides SQL-aggregated CVE data for the export API.
type CveExportView interface {
	Get(ctx context.Context, q *v1.Query) ([]CveExport, error)
	Count(ctx context.Context, q *v1.Query) (int, error)
}
