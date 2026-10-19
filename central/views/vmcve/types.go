package vmcve

import (
	"context"
	"time"

	"github.com/stackrox/rox/central/views/common"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
)

// CveCore is an interface to get VM CVE properties.
//
//go:generate mockgen-wrapper
type CveCore interface {
	GetCVE() string
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
	CountBySeverity(ctx context.Context, q *v1.Query, countOn search.FieldLabel) (common.ResourceCountByCVESeverity, error)
	Get(ctx context.Context, q *v1.Query) ([]CveCore, error)
	GetVMIDs(ctx context.Context, q *v1.Query) ([]string, error)
	GetCVEComponents(ctx context.Context, q *v1.Query) ([]CVEComponentCore, error)
	CountBySeverityPerVM(ctx context.Context, q *v1.Query) ([]VMSeverityCounts, error)
	GetAffectedVMs(ctx context.Context, q *v1.Query) ([]AffectedVMCore, error)
	CountAffectedVMs(ctx context.Context, q *v1.Query) (int, error)
	GetCVEsForVM(ctx context.Context, q *v1.Query) ([]CVEForVMCore, error)
}

// CVEForVMCore provides per-CVE aggregation scoped to a single VM.
type CVEForVMCore interface {
	GetCVE() string
	GetMaxSeverity() int32
	GetIsFixable() bool
	GetMaxCVSS() float32
	GetMaxNVDCVSS() float32
	GetEPSSProbability() float32
	GetAffectedComponentCount() int
	GetPublishDate() *time.Time
}

// VMSeverityCounts provides per-VM severity counts.
type VMSeverityCounts interface {
	GetVMID() string
	GetSeverityCounts() common.ResourceCountByCVESeverity
}

// AffectedVMCore provides per-VM aggregation for a specific CVE.
type AffectedVMCore interface {
	GetVMID() string
	GetVMName() string
	GetMaxSeverity() int32
	GetIsFixable() bool
	GetMaxCVSS() float32
	GetGuestOS() string
	GetAffectedComponentCount() int
}
