package datastore

import (
	"context"

	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
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
	scanSettingsSAC = sac.ForResource(resources.ComplianceOperator)
)

type datastoreImpl struct {
	storage       postgres.Store
	statusStorage statusStore.Store

	keyedMutex *concurrency.KeyedMutex
}

// GetScanConfiguration retrieves the scan configuration specified by name
func (ds *datastoreImpl) GetScanConfiguration(ctx context.Context, id string) (*storage.ComplianceOperatorScanSettingV2, bool, error) {
	return ds.storage.Get(ctx, id)
}

// GetScanConfigurationExists retrieves the existence of scan configuration specified by name
func (ds *datastoreImpl) GetScanConfigurationExists(ctx context.Context, scanName string) (bool, error) {
	scanConfigs, err := ds.storage.GetByQuery(ctx, search.NewQueryBuilder().
		AddExactMatches(search.ComplianceOperatorScanName, scanName).ProtoQuery())
	if err != nil {
		return false, err
	}

	return len(scanConfigs) > 0, nil
}

// GetScanConfigurations retrieves the scan configurations specified by query
func (ds *datastoreImpl) GetScanConfigurations(ctx context.Context, query *v1.Query) ([]*storage.ComplianceOperatorScanSettingV2, error) {
	return ds.storage.GetByQuery(ctx, query)
}

// UpsertScanConfiguration adds or updates the scan configuration
func (ds *datastoreImpl) UpsertScanConfiguration(ctx context.Context, scanConfig *storage.ComplianceOperatorScanSettingV2) error {
	ds.keyedMutex.Lock(scanConfig.GetId())
	defer ds.keyedMutex.Unlock(scanConfig.GetId())

	// Update the last updated time
	scanConfig.LastUpdatedTime = types.TimestampNow()
	return ds.storage.Upsert(ctx, scanConfig)
}

// DeleteScanConfiguration deletes the scan configuration specified by name
func (ds *datastoreImpl) DeleteScanConfiguration(ctx context.Context, id string) error {
	ds.keyedMutex.Lock(id)
	defer ds.keyedMutex.Unlock(id)

	return ds.storage.Delete(ctx, id)
}

// UpdateClusterStatus updates the scan configuration with the cluster status
func (ds *datastoreImpl) UpdateClusterStatus(ctx context.Context, scanID string, clusterID string, clusterStatus string) error {
	ds.keyedMutex.Lock(scanID)
	defer ds.keyedMutex.Unlock(scanID)

	// Ensure the scan configuration exists
	_, found, err := ds.GetScanConfiguration(ctx, scanID)
	if err != nil || !found {
		return errors.Errorf("Unable to find scan configuration id %q", scanID)
	}

	clusterScanStatus := &storage.ComplianceOperatorClusterScanConfigStatus{
		ClusterId: clusterID,
		ScanId:    scanID,
		Errors:    []string{clusterStatus},
	}

	return ds.statusStorage.Upsert(ctx, clusterScanStatus)
}

// GetScanConfigClusterStatus retrieves the scan configurations status per cluster specified by scan name
func (ds *datastoreImpl) GetScanConfigClusterStatus(ctx context.Context, scanID string) ([]*storage.ComplianceOperatorClusterScanConfigStatus, error) {
	return ds.statusStorage.GetByQuery(ctx, search.NewQueryBuilder().
		AddExactMatches(search.ComplianceOperatorScanConfig, scanID).ProtoQuery())
}
