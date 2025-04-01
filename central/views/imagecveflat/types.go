package imagecveflat

import (
	"context"
	"time"

	"github.com/stackrox/rox/central/views"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// CveFlat is an interface to get image CVE properties.
//
//go:generate mockgen-wrapper
type CveFlat interface {
	GetCVE() string
	GetCVEIDs() []string
	GetSeverity() *storage.VulnerabilitySeverity
	GetTopCVSS() float32
	GetTopNVDCVSS() float32
	GetAffectedImageCount() int
	GetFirstDiscoveredInSystem() *time.Time
	GetPublishDate() *time.Time
	GetFirstImageOccurrence() *time.Time
	GetState() *storage.VulnerabilityState
}

// CveFlatView interface is like a SQL view that provides functionality to fetch the image CVE data
// irrespective of the data model. One CVE can have multiple database entries if that CVE impacts multiple distros.
// Each record may have different values for properties like severity. However, the core information is the same.
// Core information such as universal CVE identifier, summary, etc. is constant.
//
//go:generate mockgen-wrapper
type CveFlatView interface {
	Count(ctx context.Context, q *v1.Query) (int, error)
	Get(ctx context.Context, q *v1.Query, options views.ReadOptions) ([]CveFlat, error)
}
