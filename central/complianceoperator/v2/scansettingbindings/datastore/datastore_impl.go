package datastore

import (
	"context"

	"github.com/stackrox/rox/central/complianceoperator/v2/scansettingbindings/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

type datastoreImpl struct {
	store postgres.Store
}

// GetScanSettingBinding retrieves the scan setting binding object from the database
func (d *datastoreImpl) GetScanSettingBinding(ctx context.Context, id string) (*storage.ComplianceOperatorScanSettingBindingV2, bool, error) {
	return d.store.Get(ctx, id)
}

// UpsertScanSettingBinding adds the scan setting binding object to the database
func (d *datastoreImpl) UpsertScanSettingBinding(ctx context.Context, scanSettingBinding *storage.ComplianceOperatorScanSettingBindingV2) error {
	return d.store.Upsert(ctx, scanSettingBinding)
}

// DeleteScan removes a scan setting binding object from the database
func (d *datastoreImpl) DeleteScanSettingBinding(ctx context.Context, id string) error {
	return d.store.Delete(ctx, id)
}

// GetScanSettingBindingsByCluster retrieves scan setting bindings by cluster
func (d *datastoreImpl) GetScanSettingBindingsByCluster(ctx context.Context, clusterID string) ([]*storage.ComplianceOperatorScanSettingBindingV2, error) {
	return d.store.GetByQuery(ctx, search.NewQueryBuilder().
		AddExactMatches(search.ClusterID, clusterID).ProtoQuery())
}
