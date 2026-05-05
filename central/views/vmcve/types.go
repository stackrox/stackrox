package vmcve

import (
	"context"
	"time"

	"github.com/stackrox/rox/central/views/common"
	v1 "github.com/stackrox/rox/generated/api/v1"
)

// CveCore is an interface to get VM CVE properties.
//
//go:generate mockgen-wrapper
type CveCore interface {
	GetCVE() string
	GetCVEIDs() []string
	GetVMsBySeverity() common.ResourceCountByCVESeverity
	GetTopCVSS() float32
	GetAffectedVMCount() int
	GetFirstDiscoveredInSystem() *time.Time
	GetPublishDate() *time.Time
	GetEPSSProbability() float32
}

// CVEComponentCore provides component details for a specific CVE.
//
//go:generate mockgen-wrapper
type CVEComponentCore interface {
	GetComponentName() string
	GetComponentVersion() string
	GetComponentSource() int32
	GetFixedBy() string
	GetAdvisoryName() string
	GetAdvisoryLink() string
}

// CveView interface is like a SQL view that provides functionality to fetch VM CVE data
// irrespective of the data model. One CVE can have multiple database entries if that CVE
// impacts multiple VMs or components. However, the core information is the same.
//
//go:generate mockgen-wrapper
type CveView interface {
	Count(ctx context.Context, q *v1.Query) (int, error)
	CountBySeverity(ctx context.Context, q *v1.Query) (common.ResourceCountByCVESeverity, error)
	Get(ctx context.Context, q *v1.Query) ([]CveCore, error)
	GetVMIDs(ctx context.Context, q *v1.Query) ([]string, error)
	GetCVEComponents(ctx context.Context, q *v1.Query) ([]CVEComponentCore, error)
	CountBySeverityPerVM(ctx context.Context, q *v1.Query) ([]VMSeverityCounts, error)
}

// VMSeverityCounts provides per-VM severity counts.
type VMSeverityCounts interface {
	GetVMID() string
	GetSeverityCounts() common.ResourceCountByCVESeverity
}
