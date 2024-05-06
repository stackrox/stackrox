package datastore

import (
	"context"

	"github.com/stackrox/rox/central/complianceoperator/v2/scansettingbindings/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
)

var (
	complianceSAC = sac.ForResource(resources.Compliance)
)

type datastoreImpl struct {
	store postgres.Store
}

// GetScanSettingBindings retrieves scan setting bindings matching the query
func (d *datastoreImpl) GetScanSettingBindings(ctx context.Context, query *v1.Query) ([]*storage.ComplianceOperatorScanSettingBindingV2, error) {
	bindings, err := d.store.GetByQuery(ctx, query)

	// SAC will return a config if a user has permissions to ANY of the clusters.  For tech preview, and
	// in the interest of ensuring we don't leak clusters, if a user does not have access to one or more
	// of the clusters returned by the query, we will return nothing.  An all or nothing approach in the
	// interest of not leaking data.
	// We must ensure the user has access to all the clusters in a config.  The SAC filter will return the row
	// if the user has access to any cluster
	if !complianceSAC.ScopeChecker(ctx, storage.Access_READ_ACCESS).AllAllowed(getScopeKeys(bindings)) {
		return nil, nil
	}

	return bindings, err
}

// GetScanSettingBinding retrieves the scan setting binding object from the database
func (d *datastoreImpl) GetScanSettingBinding(ctx context.Context, id string) (*storage.ComplianceOperatorScanSettingBindingV2, bool, error) {
	return d.store.Get(ctx, id)
}

// UpsertScanSettingBinding adds the scan setting binding object to the database
func (d *datastoreImpl) UpsertScanSettingBinding(ctx context.Context, scanSettingBinding *storage.ComplianceOperatorScanSettingBindingV2) error {
	return d.store.Upsert(ctx, scanSettingBinding)
}

// DeleteScanSettingBinding removes a scan setting binding object from the database
func (d *datastoreImpl) DeleteScanSettingBinding(ctx context.Context, id string) error {
	return d.store.Delete(ctx, id)
}

// GetScanSettingBindingsByCluster retrieves scan setting bindings by cluster
func (d *datastoreImpl) GetScanSettingBindingsByCluster(ctx context.Context, clusterID string) ([]*storage.ComplianceOperatorScanSettingBindingV2, error) {
	return d.store.GetByQuery(ctx, search.NewQueryBuilder().
		AddExactMatches(search.ClusterID, clusterID).ProtoQuery())
}

func getScopeKeys(bindings []*storage.ComplianceOperatorScanSettingBindingV2) [][]sac.ScopeKey {
	clusterScopeKeys := make([][]sac.ScopeKey, 0, len(bindings))
	for _, binding := range bindings {
		clusterScopeKeys = append(clusterScopeKeys, []sac.ScopeKey{sac.ClusterScopeKey(binding.GetClusterId())})
	}
	return clusterScopeKeys
}

// DeleteScanSettingByCluster deletes scan setting by cluster
func (d *datastoreImpl) DeleteScanSettingByCluster(ctx context.Context, clusterID string) error {
	query := search.NewQueryBuilder().AddStrings(search.ClusterID, clusterID).ProtoQuery()
	_, err := d.store.DeleteByQuery(ctx, query)
	if err != nil {
		return err
	}
	return nil

}
