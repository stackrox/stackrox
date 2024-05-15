package platformcve

import (
	"context"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// CveCore is an interface to get platform CVE properties.
//
//go:generate mockgen-wrapper
type CveCore interface {
	GetCVE() string
	GetCVEID() string
	GetCVEType() storage.CVE_CVEType
	GetCVSS() float32
	GetClusterCount() int
	GetClusterCountByPlatformType() ClusterCountByPlatformType
	GetFixability() bool
	GetFirstDiscoveredTime() *time.Time
}

// CveView interface is like a SQL view that provides the functionality to fetch platform CVE data
//
//go:generate mockgen-wrapper
type CveView interface {
	Count(ctx context.Context, q *v1.Query) (int, error)
	Get(ctx context.Context, q *v1.Query) ([]CveCore, error)
	GetClusterIDs(ctx context.Context, q *v1.Query) ([]string, error)
}

// ClusterCountByPlatformType provides ability to retrieve number of clusters of each platform type
type ClusterCountByPlatformType interface {
	GetGenericClusterCount() int
	GetKubernetesClusterCount() int
	GetOpenshiftClusterCount() int
	GetOpenshift4ClusterCount() int
}
