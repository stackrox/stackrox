package datastore

import (
	"context"

	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	clusterDatastore "github.com/stackrox/rox/central/cluster/datastore"
	statusStore "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/scanconfigstatus/store/postgres"
	"github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/sliceutils"
	"github.com/stackrox/rox/pkg/uuid"
)

var (
	complianceSAC = sac.ForResource(resources.Compliance)
)

type datastoreImpl struct {
	storage       postgres.Store
	statusStorage statusStore.Store
	clusterDS     clusterDatastore.DataStore
	keyedMutex    *concurrency.KeyedMutex
}

// GetScanConfiguration retrieves the scan configuration specified by id
func (ds *datastoreImpl) GetScanConfiguration(ctx context.Context, id string) (*storage.ComplianceOperatorScanConfigurationV2, bool, error) {
	scanConfig, found, err := ds.storage.Get(ctx, id)

	// We must ensure the user has access to all the clusters in a config.  The SAC filter will return the row
	// if the user has access to any cluster
	if !complianceSAC.ScopeChecker(ctx, storage.Access_READ_ACCESS).AllAllowed(getScopeKeys(scanConfig.GetClusters())) {
		return nil, false, nil
	}

	return scanConfig, found, err
}

// ScanConfigurationExists retrieves the existence of scan configuration specified by name
func (ds *datastoreImpl) ScanConfigurationExists(ctx context.Context, scanName string) (bool, error) {
	scanConfigs, err := ds.storage.GetByQuery(ctx, search.NewQueryBuilder().
		AddExactMatches(search.ComplianceOperatorScanConfigName, scanName).ProtoQuery())
	if err != nil {
		return false, err
	}

	return len(scanConfigs) > 0, nil
}

// GetScanConfigurations retrieves the scan configurations specified by query
func (ds *datastoreImpl) GetScanConfigurations(ctx context.Context, query *v1.Query) ([]*storage.ComplianceOperatorScanConfigurationV2, error) {
	scanConfigs, err := ds.storage.GetByQuery(ctx, query)

	// SAC will return a config if a user has permissions to ANY of the clusters.  For tech preview, and
	// in the interest of ensuring we don't leak clusters, if a user does not have access to one or more
	// of the clusters returned by the query, we will return nothing.  An all or nothing approach in the
	// interest of not leaking data.
	for _, scanConfig := range scanConfigs {
		// We must ensure the user has access to all the clusters in a config.  The SAC filter will return the row
		// if the user has access to any cluster
		if !complianceSAC.ScopeChecker(ctx, storage.Access_READ_ACCESS).AllAllowed(getScopeKeys(scanConfig.GetClusters())) {
			return nil, nil
		}
	}

	return scanConfigs, err
}

// UpsertScanConfiguration adds or updates the scan configuration
func (ds *datastoreImpl) UpsertScanConfiguration(ctx context.Context, scanConfig *storage.ComplianceOperatorScanConfigurationV2) error {
	// SAC for an upsert requires access to all clusters present in the conifg.  This is handled
	// in the store so a SAC check is not needed here.

	ds.keyedMutex.Lock(scanConfig.GetId())
	defer ds.keyedMutex.Unlock(scanConfig.GetId())

	// Update the last updated time
	scanConfig.LastUpdatedTime = types.TimestampNow()
	return ds.storage.Upsert(ctx, scanConfig)
}

// UpsertNoLockScanConfiguration upserts scan config like UpsertScanConfiguration but does not create a lock
func (ds *datastoreImpl) UpsertNoLockScanConfiguration(ctx context.Context, scanConfig *storage.ComplianceOperatorScanConfigurationV2) error {
	scanConfig.LastUpdatedTime = types.TimestampNow()
	return ds.storage.Upsert(ctx, scanConfig)
}

// DeleteScanConfiguration deletes the scan configuration specified by id
func (ds *datastoreImpl) DeleteScanConfiguration(ctx context.Context, id string) (string, error) {
	// Need to verify that write to all clusters used in this configuration is allowed.
	elevatedSACReadCtx := sac.WithAllAccess(context.Background())

	// Use elevated privileges to get all clusters associated with this configuration.
	scanConfig, found, err := ds.GetScanConfiguration(elevatedSACReadCtx, id)
	if err != nil {
		return "", errors.Wrapf(err, "Unable to find scan configuration id %q", id)
	}
	if !found {
		return "", errors.Errorf("Scan configuration id %q not found", id)
	}
	scanConfigName := scanConfig.GetScanConfigName()

	if !complianceSAC.ScopeChecker(ctx, storage.Access_READ_WRITE_ACCESS).AllAllowed(getScopeKeys(scanConfig.GetClusters())) {
		return "", sac.ErrResourceAccessDenied
	}

	ds.keyedMutex.Lock(id)
	defer ds.keyedMutex.Unlock(id)

	// remove scan data from scan status table first
	_, err = ds.statusStorage.DeleteByQuery(ctx, search.NewQueryBuilder().
		AddExactMatches(search.ComplianceOperatorScanConfig, id).ProtoQuery())
	if err != nil {
		return "", errors.Wrapf(err, "Unable to delete scan status for scan configuration id %q", id)
	}

	err = ds.storage.Delete(ctx, id)
	if err != nil {
		return "", errors.Wrapf(err, "Unable to delete scan configuration id %q", id)
	}

	return scanConfigName, nil
}

// UpdateClusterStatus updates the scan configuration with the cluster status
func (ds *datastoreImpl) UpdateClusterStatus(ctx context.Context, scanConfigID string, clusterID string, clusterStatus string) error {
	if !complianceSAC.ScopeChecker(ctx, storage.Access_READ_WRITE_ACCESS).IsAllowed(sac.ClusterScopeKey(clusterID)) {
		return sac.ErrResourceAccessDenied
	}

	// Look up the cluster, so we can store the name for convenience AND history
	cluster, exists, err := ds.clusterDS.GetCluster(ctx, clusterID)
	if err != nil {
		return err
	}
	if !exists {
		return errors.Errorf("could not pull config for cluster %q because it does not exist", clusterID)
	}

	ds.keyedMutex.Lock(scanConfigID)
	defer ds.keyedMutex.Unlock(scanConfigID)

	// Ensure the scan configuration exists
	_, found, err := ds.GetScanConfiguration(ctx, scanConfigID)
	if err != nil {
		return errors.Wrapf(err, "Unable to retrieve scan configuration id %q", scanConfigID)
	}
	if !found {
		return errors.Errorf("Unable to find scan configuration id %q", scanConfigID)
	}

	// Need to build a deterministic ID from clusterID and scanID to ensure we always have the latest status
	clusterUUID, err := uuid.FromString(clusterID)
	if err != nil {
		return errors.Wrapf(err, "Unable to build scan configuration status id based off %q", scanConfigID)
	}
	statusKey := uuid.NewV5(clusterUUID, scanConfigID).String()

	clusterScanStatus := &storage.ComplianceOperatorClusterScanConfigStatus{
		Id:           statusKey,
		ClusterId:    clusterID,
		ClusterName:  cluster.GetName(),
		ScanConfigId: scanConfigID,
		Errors:       []string{clusterStatus},
	}

	return ds.statusStorage.Upsert(ctx, clusterScanStatus)
}

// GetScanConfigClusterStatus retrieves the scan configurations status per cluster specified by scan id
func (ds *datastoreImpl) GetScanConfigClusterStatus(ctx context.Context, scanConfigID string) ([]*storage.ComplianceOperatorClusterScanConfigStatus, error) {
	return ds.statusStorage.GetByQuery(ctx, search.NewQueryBuilder().
		AddExactMatches(search.ComplianceOperatorScanConfig, scanConfigID).ProtoQuery())
}

func (ds *datastoreImpl) CountScanConfigurations(ctx context.Context, q *v1.Query) (int, error) {
	// Need to account for cluster SAC, so first get the configs with the SAC filters applied
	scanConfigs, err := ds.GetScanConfigurations(ctx, q)
	return len(scanConfigs), err
}

func getScopeKeys(scanClusters []*storage.ComplianceOperatorScanConfigurationV2_Cluster) [][]sac.ScopeKey {
	clusterScopeKeys := make([][]sac.ScopeKey, 0, len(scanClusters))
	for _, scanCluster := range scanClusters {
		clusterScopeKeys = append(clusterScopeKeys, []sac.ScopeKey{sac.ClusterScopeKey(scanCluster.GetClusterId())})
	}

	return clusterScopeKeys
}

func (ds *datastoreImpl) RemoveClusterFromScanConfig(ctx context.Context, clusterID string) error {
	q := search.NewQueryBuilder().
		AddExactMatches(search.ClusterID, clusterID).ProtoQuery()
	scans, err := ds.GetScanConfigurations(ctx, q)
	if err != nil {
		return err
	}
	for _, scan := range scans {
		ds.keyedMutex.Lock(scan.GetId())
		clusters := scan.GetClusters()
		filterFunction := func(cluster *storage.ComplianceOperatorScanConfigurationV2_Cluster) bool {
			return cluster.GetClusterId() != clusterID
		}
		newClusters := sliceutils.Filter(clusters, filterFunction)
		scan.Clusters = newClusters
		err := ds.UpsertNoLockScanConfiguration(ctx, scan)
		if err != nil {
			return err
		}
		ds.keyedMutex.Unlock(scan.GetId())
		_, err = ds.statusStorage.DeleteByQuery(ctx, search.NewQueryBuilder().
			AddExactMatches(search.ComplianceOperatorScanConfig, scan.GetId()).ProtoQuery())
		if err != nil {
			return errors.Wrapf(err, "Unable to delete scan status for scan configuration id %q", scan.GetId())
		}
	}
	return nil
}
