package datastore

import (
	"context"

	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	clusterDatastore "github.com/stackrox/rox/central/cluster/datastore"
	scanConfigSearch "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/datastore/search"
	statusStore "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/scanconfigstatus/store/postgres"
	"github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
)

// TODO(ROX-19742):  Figure out SAC for the configurations
var (
	scanConfigurationsSAC = sac.ForResource(resources.ComplianceOperator)
)

type datastoreImpl struct {
	storage       postgres.Store
	statusStorage statusStore.Store
	clusterDS     clusterDatastore.DataStore
	keyedMutex    *concurrency.KeyedMutex
	searcher      scanConfigSearch.Searcher
}

// GetScanConfiguration retrieves the scan configuration specified by id
func (ds *datastoreImpl) GetScanConfiguration(ctx context.Context, id string) (*storage.ComplianceOperatorScanConfigurationV2, bool, error) {
	if ok, err := scanConfigurationsSAC.ReadAllowed(ctx); err != nil {
		return nil, false, err
	} else if !ok {
		return nil, false, nil
	}

	return ds.storage.Get(ctx, id)
}

// ScanConfigurationExists retrieves the existence of scan configuration specified by name
func (ds *datastoreImpl) ScanConfigurationExists(ctx context.Context, scanName string) (bool, error) {
	if ok, err := scanConfigurationsSAC.ReadAllowed(ctx); err != nil {
		return false, err
	} else if !ok {
		return false, nil
	}

	scanConfigs, err := ds.storage.GetByQuery(ctx, search.NewQueryBuilder().
		AddExactMatches(search.ComplianceOperatorScanName, scanName).ProtoQuery())
	if err != nil {
		return false, err
	}

	return len(scanConfigs) > 0, nil
}

// GetScanConfigurations retrieves the scan configurations specified by query
func (ds *datastoreImpl) GetScanConfigurations(ctx context.Context, query *v1.Query) ([]*storage.ComplianceOperatorScanConfigurationV2, error) {
	if ok, err := scanConfigurationsSAC.ReadAllowed(ctx); err != nil {
		return nil, err
	} else if !ok {
		return nil, nil
	}

	return ds.storage.GetByQuery(ctx, query)
}

// UpsertScanConfiguration adds or updates the scan configuration
func (ds *datastoreImpl) UpsertScanConfiguration(ctx context.Context, scanConfig *storage.ComplianceOperatorScanConfigurationV2) error {
	if ok, err := scanConfigurationsSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	ds.keyedMutex.Lock(scanConfig.GetId())
	defer ds.keyedMutex.Unlock(scanConfig.GetId())

	// Update the last updated time
	scanConfig.LastUpdatedTime = types.TimestampNow()
	return ds.storage.Upsert(ctx, scanConfig)
}

// DeleteScanConfiguration deletes the scan configuration specified by id
func (ds *datastoreImpl) DeleteScanConfiguration(ctx context.Context, id string) error {
	if ok, err := scanConfigurationsSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	ds.keyedMutex.Lock(id)
	defer ds.keyedMutex.Unlock(id)

	return ds.storage.Delete(ctx, id)
}

// UpdateClusterStatus updates the scan configuration with the cluster status
func (ds *datastoreImpl) UpdateClusterStatus(ctx context.Context, scanID string, clusterID string, clusterStatus string) error {
	if ok, err := scanConfigurationsSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
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

	ds.keyedMutex.Lock(scanID)
	defer ds.keyedMutex.Unlock(scanID)

	// Ensure the scan configuration exists
	_, found, err := ds.GetScanConfiguration(ctx, scanID)
	if err != nil || !found {
		return errors.Errorf("Unable to find scan configuration id %q", scanID)
	}

	clusterScanStatus := &storage.ComplianceOperatorClusterScanConfigStatus{
		ClusterId:   clusterID,
		ClusterName: cluster.GetName(),
		ScanId:      scanID,
		Errors:      []string{clusterStatus},
	}

	return ds.statusStorage.Upsert(ctx, clusterScanStatus)
}

// GetScanConfigClusterStatus retrieves the scan configurations status per cluster specified by scan id
func (ds *datastoreImpl) GetScanConfigClusterStatus(ctx context.Context, scanID string) ([]*storage.ComplianceOperatorClusterScanConfigStatus, error) {
	if ok, err := scanConfigurationsSAC.ReadAllowed(ctx); err != nil {
		return nil, err
	} else if !ok {
		return nil, nil
	}

	return ds.statusStorage.GetByQuery(ctx, search.NewQueryBuilder().
		AddExactMatches(search.ComplianceOperatorScanConfig, scanID).ProtoQuery())
}

func (ds *datastoreImpl) CountScanConfigurations(ctx context.Context, q *v1.Query) (int, error) {
	return ds.searcher.Count(ctx, q)
}
