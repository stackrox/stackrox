package datastore

import (
	"context"

	"github.com/stackrox/rox/central/complianceoperator/v2/scans/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

type datastoreImpl struct {
	store postgres.Store
}

// GetScan retrieves the scan object from the database
func (d *datastoreImpl) GetScan(ctx context.Context, id string) (*storage.ComplianceOperatorScanV2, bool, error) {
	return d.store.Get(ctx, id)
}

// UpsertScan adds the scan object to the database
func (d *datastoreImpl) UpsertScan(ctx context.Context, scan *storage.ComplianceOperatorScanV2) error {
	return d.store.Upsert(ctx, scan)
}

// DeleteScan removes a scan object from the database
func (d *datastoreImpl) DeleteScan(ctx context.Context, id string) error {
	return d.store.Delete(ctx, id)
}

// GetScansByCluster retrieves scan objects by cluster
func (d *datastoreImpl) GetScansByCluster(ctx context.Context, clusterID string) ([]*storage.ComplianceOperatorScanV2, error) {
	return d.store.GetByQuery(ctx, search.NewQueryBuilder().
		AddExactMatches(search.ClusterID, clusterID).ProtoQuery())
}
