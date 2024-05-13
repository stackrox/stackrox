package nodecve

import (
	"context"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
)

// CveCore is an interface to get node CVE properties.
//
//go:generate mockgen-wrapper
type CveCore interface {
	GetCVE() string
	GetCVEIDs() []string
	GetTopCVSS() float32
	GetNodeCount() int
	GetNodeCountBySeverity() ResourceCountByCVESeverity
	GetNodeIDs() []string
	GetFirstDiscoveredTime() *time.Time
	GetOperatingSystemCount() int
}

// CveView interface is like a SQL view that provides the functionality to fetch node CVE data
// irrespective of the data model. One CVE can have multiple database entries if that CVE impacts multiple distros.
// Each record may have different values for properties like severity. However, the core information is the same.
// Core information such as universal CVE identifier, summary, etc. is constant.
//
//go:generate mockgen-wrapper
type CveView interface {
	Count(ctx context.Context, q *v1.Query) (int, error) // NodeCVECount
	Get(ctx context.Context, q *v1.Query) ([]CveCore, error)
	CountBySeverity(ctx context.Context, q *v1.Query) (ResourceCountByCVESeverity, error) // node view count cve by severity
	GetNodeIDs(ctx context.Context, q *v1.Query) ([]string, error)
}

// ResourceCountByCVESeverity provides ability to retrieve number of resources by severity
// TODO(cdu): unify this interface with common.ResourceCountByCVESeverity maybe after platform merge.
type ResourceCountByCVESeverity interface {
	GetCriticalSeverityCount() int
	GetImportantSeverityCount() int
	GetModerateSeverityCount() int
	GetLowSeverityCount() int
}
