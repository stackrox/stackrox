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

// UpsertScan adds the scan object to the database.  If enabling the use of this
// method from a service, the creation of the `ProfileRefID` and `ScanRefID` must be accounted for.  In reality this
// method should only be used by the pipeline as this is a compliance operator object we are storing.
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

// DeleteScanByCluster deletes scans by cluster
func (d *datastoreImpl) DeleteScanByCluster(ctx context.Context, clusterID string) error {
	query := search.NewQueryBuilder().AddStrings(search.ClusterID, clusterID).ProtoQuery()
	_, err := d.store.DeleteByQuery(ctx, query)
	if err != nil {
		return err
	}
	return nil
}
