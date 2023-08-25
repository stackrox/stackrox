package imagecve

import (
	"context"
	"time"

	"github.com/stackrox/rox/central/views"
	"github.com/stackrox/rox/central/views/common"
	v1 "github.com/stackrox/rox/generated/api/v1"
)

// CveCore is an interface to get image CVE properties.
//
//go:generate mockgen-wrapper
type CveCore interface {
	GetCVE() string
	GetCVEIDs() []string
	GetImagesBySeverity() common.ResourceCountByCVESeverity
	GetTopCVSS() float32
	GetAffectedImageCount() int
	GetFirstDiscoveredInSystem() *time.Time
}

// CveView interface is like a SQL view that provides functionality to fetch the image CVE data
// irrespective of the data model. One CVE can have multiple database entries if that CVE impacts multiple distros.
// Each record may have different values for properties like severity. However, the core information is the same.
// Core information such as universal CVE identifier, summary, etc. is constant.
//
//go:generate mockgen-wrapper
type CveView interface {
	Count(ctx context.Context, q *v1.Query) (int, error)
	CountBySeverity(ctx context.Context, q *v1.Query) (common.ResourceCountByCVESeverity, error)
	Get(ctx context.Context, q *v1.Query, options views.ReadOptions) ([]CveCore, error)
	GetImageIDs(ctx context.Context, q *v1.Query) ([]string, error)
	GetDeploymentIDs(ctx context.Context, q *v1.Query) ([]string, error)
}
