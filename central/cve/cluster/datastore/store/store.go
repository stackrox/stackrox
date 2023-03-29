package store

import (
	"context"

	"github.com/stackrox/rox/central/cve/converter/v2"
	"github.com/stackrox/rox/generated/storage"
)

// Store provides storage functionality for CVEs.
//
//go:generate mockgen-wrapper
type Store interface {
	Count(ctx context.Context) (int, error)
	Exists(ctx context.Context, id string) (bool, error)

	Get(ctx context.Context, id string) (*storage.ClusterCVE, bool, error)
	GetMany(ctx context.Context, ids []string) ([]*storage.ClusterCVE, []int, error)

	UpsertMany(ctx context.Context, cves []*storage.ClusterCVE) error

	ReconcileClusterCVEParts(ctx context.Context, cveType storage.CVE_CVEType, cvePartsArr ...converter.ClusterCVEParts) error
	DeleteClusterCVEsForCluster(ctx context.Context, clusterID string) error
}
